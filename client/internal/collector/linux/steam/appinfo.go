package steam

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strconv"
)

// Steam's cache of store metadata, read for the four grid pictures. A title
// published from roughly 2024 on stores them behind a hashed path segment that
// cannot be derived from anything else, so without this file new games get no hero
// and no logo and the card degrades for no reason.

const (
	appInfoMagic              = 0x07564400 // 'VDF' plus a version byte
	appInfoMagicMask          = 0xffffff00
	appInfoStringTableVersion = 0x29
)

type assets struct {
	header  string
	capsule string
	hero    string
	logo    string
}

func (a assets) empty() bool {
	return a.header == "" && a.capsule == "" && a.hero == "" && a.logo == ""
}

// appInfoRecord returns one app's payload. The file is a flat table - appid, size,
// then that many bytes, ended by a bare zero appid - so a payload is reachable
// without decoding the binary KeyValues inside it. Version 0x29 moved the keys into
// a string table after the records and left the table itself alone, which is what
// makes this safe across versions.
func appInfoRecord(data []byte, appID int) ([]byte, bool) {
	if len(data) < 16 {
		return nil, false
	}
	magic := binary.LittleEndian.Uint32(data)
	if magic&appInfoMagicMask != appInfoMagic {
		return nil, false
	}

	off, end := 8, len(data)
	if magic&0xff >= appInfoStringTableVersion {
		table := int(binary.LittleEndian.Uint64(data[8:]))
		if table <= 16 || table > len(data) {
			return nil, false
		}
		off, end = 16, table
	}

	for off+4 <= end {
		id := int(binary.LittleEndian.Uint32(data[off:]))
		if id == 0 || off+8 > end {
			return nil, false
		}
		size := int(binary.LittleEndian.Uint32(data[off+4:]))
		if size <= 0 || off+8+size > end {
			return nil, false
		}
		if id == appID {
			return data[off+8 : off+8+size], true
		}
		off += 8 + size
	}
	return nil, false
}

// scanAssets picks the picture names out of a payload. In 0x29 the keys are string
// table indices and would have to be resolved to be read, but the values name
// themselves. A language or size variant does not match and is skipped.
func scanAssets(blob []byte) assets {
	var out assets
	for _, tok := range bytes.Split(blob, []byte{0}) {
		if len(tok) < len("logo.png") || len(tok) > 72 || !printableASCII(tok) {
			continue
		}
		name := tok
		if i := bytes.LastIndexByte(tok, '/'); i >= 0 {
			if !hexDir(tok[:i]) {
				continue
			}
			name = tok[i+1:]
		}

		var field *string
		switch string(name) {
		case "header.jpg":
			field = &out.header
		case "library_600x900.jpg", "library_capsule.jpg": // the old and new portrait names
			field = &out.capsule
		case "library_hero.jpg":
			field = &out.hero
		case "logo.png":
			field = &out.logo
		default:
			continue
		}
		if *field == "" {
			*field = string(tok)
		}
	}
	return out
}

func printableASCII(b []byte) bool {
	for _, c := range b {
		if c < 32 || c >= 127 {
			return false
		}
	}
	return true
}

func hexDir(b []byte) bool {
	if len(b) != 40 {
		return false
	}
	for _, c := range b {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// One prefix serves both the bare and the hashed form.
const assetBase = "https://shared.akamai.steamstatic.com/store_item_assets/steam/apps/"

func assetURL(appID int, path string) string {
	if path == "" {
		return ""
	}
	return assetBase + strconv.Itoa(appID) + "/" + path
}

// The file is a megabyte and a half, so this runs on the resolver's goroutine and
// its result goes to our own cache rather than being re-read per tick.
func localAssets(root string, appID int) (assets, bool) {
	data, err := os.ReadFile(filepath.Join(root, "appcache", "appinfo.vdf"))
	if err != nil {
		return assets{}, false
	}
	blob, ok := appInfoRecord(data, appID)
	if !ok {
		return assets{}, false
	}
	a := scanAssets(blob)
	return a, !a.empty()
}
