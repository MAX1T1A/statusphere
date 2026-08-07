package steam

import "strings"

// Valve's KeyValues text format. Both files this package reads are shallow, so
// there is no tree here: scanKV reports the depth a pair was found at and callers
// take what they want from the depth they want. A "name" inside InstalledDepots is
// a different key from AppState.name.

func scanKV(data []byte, fn func(depth int, key, value string)) {
	depth := 0
	pending := ""
	have := false

	for i := 0; i < len(data); {
		switch c := data[i]; {
		case c == '{':
			// The token before the brace named this object, it was not a value
			depth, have, i = depth+1, false, i+1
		case c == '}':
			depth, have, i = depth-1, false, i+1
		case c == '/' && i+1 < len(data) && data[i+1] == '/':
			for i < len(data) && data[i] != '\n' {
				i++
			}
		case c == '"':
			tok, next, ok := readQuoted(data, i)
			if !ok {
				return // truncated file: keep what was read
			}
			i = next
			if have {
				fn(depth, pending, tok)
				have = false
			} else {
				pending, have = tok, true
			}
		default:
			i++
		}
	}
}

func readQuoted(data []byte, i int) (string, int, bool) {
	var b strings.Builder
	for j := i + 1; j < len(data); j++ {
		switch data[j] {
		case '\\':
			if j+1 >= len(data) {
				return "", 0, false
			}
			j++
			switch data[j] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			default:
				b.WriteByte(data[j])
			}
		case '"':
			return b.String(), j + 1, true
		default:
			b.WriteByte(data[j])
		}
	}
	return "", 0, false
}

// parseFlat returns the scalars directly inside the root object. Keys are
// lower-cased: KeyValues is case-insensitive and an appmanifest mixes "appid"
// with "LastPlayed".
func parseFlat(data []byte) map[string]string {
	out := make(map[string]string, 24)
	scanKV(data, func(depth int, k, v string) {
		if depth == 1 {
			out[strings.ToLower(k)] = v
		}
	})
	return out
}

// libraryPaths returns the library roots in a libraryfolders.vdf. They sit at
// depth 2; the "apps" block under each holds sizes keyed by appid and would look
// like more roots one level down.
func libraryPaths(data []byte) []string {
	var out []string
	scanKV(data, func(depth int, k, v string) {
		if depth == 2 && strings.EqualFold(k, "path") && v != "" {
			out = append(out, v)
		}
	})
	return out
}
