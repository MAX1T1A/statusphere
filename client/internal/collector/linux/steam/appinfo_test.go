package steam

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// buildAppInfo assembles a version 0x29 file: magic, universe, the string table
// offset, then records, then the bare zero appid that ends them.
func buildAppInfo(records map[int]string, order []int) []byte {
	var body bytes.Buffer
	for _, id := range order {
		payload := []byte(records[id])
		binary.Write(&body, binary.LittleEndian, uint32(id))
		binary.Write(&body, binary.LittleEndian, uint32(len(payload)))
		body.Write(payload)
	}
	binary.Write(&body, binary.LittleEndian, uint32(0))

	var out bytes.Buffer
	binary.Write(&out, binary.LittleEndian, uint32(0x07564429))
	binary.Write(&out, binary.LittleEndian, uint32(1))
	binary.Write(&out, binary.LittleEndian, uint64(16+body.Len()))
	out.Write(body.Bytes())
	out.WriteString("the string table") // must not be walked as a record
	return out.Bytes()
}

func TestAppInfoRecord(t *testing.T) {
	data := buildAppInfo(map[int]string{
		1174180: "first payload",
		2183900: "second payload",
	}, []int{1174180, 2183900})

	for appID, want := range map[int]string{1174180: "first payload", 2183900: "second payload"} {
		got, ok := appInfoRecord(data, appID)
		if !ok || string(got) != want {
			t.Errorf("appid %d: got %q ok=%v, want %q", appID, got, ok, want)
		}
	}
	if _, ok := appInfoRecord(data, 99); ok {
		t.Error("found an app that is not in the file")
	}
}

func TestAppInfoRecordRejectsJunk(t *testing.T) {
	good := buildAppInfo(map[int]string{1: "x"}, []int{1})

	cases := map[string][]byte{
		"empty":         {},
		"short":         good[:12],
		"wrong magic":   append([]byte{0xde, 0xad, 0xbe, 0xef}, good[4:]...),
		"truncated mid": good[:len(good)-12],
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			if _, ok := appInfoRecord(in, 1); ok && name != "truncated mid" {
				t.Error("claimed a record in junk")
			}
		})
	}
}

func TestScanAssetsBareNames(t *testing.T) {
	blob := bytes.Join([][]byte{
		[]byte("header.jpg"),
		[]byte("library_600x900.jpg"),
		[]byte("library_hero.jpg"),
		[]byte("logo.png"),
		[]byte("capsule_231x87.jpg"), // a picture we do not use
		[]byte("library_hero_blur.jpg"),
	}, []byte{0})

	got := scanAssets(blob)
	want := assets{header: "header.jpg", capsule: "library_600x900.jpg", hero: "library_hero.jpg", logo: "logo.png"}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestScanAssetsHashedNames(t *testing.T) {
	// The real strings out of appid 3725100, plus the variants that must not win
	blob := bytes.Join([][]byte{
		[]byte("d77c3a71a30916289998c93f32e08dd94c5a783f/header.jpg"),
		[]byte("d77c3a71a30916289998c93f32e08dd94c5a783f/header_spanish.jpg"),
		[]byte("9e37c4fe713653b8a4f6356dd0380a9b844d515f/library_capsule.jpg"),
		[]byte("b9647d915f892d34efcb52cf1970af597acc84c7/library_hero.jpg"),
		[]byte("ff4189156a473cc7705af59011f94976cb648a88/logo.png"),
		[]byte("notahash/logo.png"), // a path segment that is not a 40-hex digest
	}, []byte{0})

	got := scanAssets(blob)
	want := assets{
		header:  "d77c3a71a30916289998c93f32e08dd94c5a783f/header.jpg",
		capsule: "9e37c4fe713653b8a4f6356dd0380a9b844d515f/library_capsule.jpg",
		hero:    "b9647d915f892d34efcb52cf1970af597acc84c7/library_hero.jpg",
		logo:    "ff4189156a473cc7705af59011f94976cb648a88/logo.png",
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestScanAssetsEmpty(t *testing.T) {
	if got := scanAssets([]byte("nothing here at all")); !got.empty() {
		t.Errorf("got %+v, want nothing", got)
	}
}

func TestAssetURL(t *testing.T) {
	cases := map[string]string{
		"header.jpg": assetBase + "1174180/header.jpg",
		"d77c3a71a30916289998c93f32e08dd94c5a783f/header.jpg": assetBase + "1174180/d77c3a71a30916289998c93f32e08dd94c5a783f/header.jpg",
		"": "",
	}
	for in, want := range cases {
		if got := assetURL(1174180, in); got != want {
			t.Errorf("assetURL(%q) = %q, want %q", in, got, want)
		}
	}
}
