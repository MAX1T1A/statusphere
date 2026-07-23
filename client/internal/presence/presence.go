package presence

import (
	"encoding/json"
	"maps"
	"reflect"
)

type Snapshot map[string]any

func New() Snapshot {
	return make(Snapshot)
}

func (s Snapshot) Clone() Snapshot {
	out := make(Snapshot, len(s))
	maps.Copy(out, s)
	return out
}

func (s Snapshot) Has(key string) bool {
	_, ok := s[key]
	return ok
}

func (s Snapshot) Set(key string, value any) {
	s[key] = value
}

func (s Snapshot) String(key string) string {
	v, _ := s[key].(string)
	return v
}

func (s Snapshot) Float(key string) (float64, bool) {
	switch v := s[key].(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func (s Snapshot) Int(key string) (int64, bool) {
	switch v := s[key].(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case float64:
		return int64(v), true
	case float32:
		return int64(v), true
	case json.Number:
		n, err := v.Int64()
		return n, err == nil
	default:
		return 0, false
	}
}

func (s Snapshot) Strings(key string) []string {
	switch v := s[key].(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok && str != "" {
				out = append(out, str)
			}
		}
		return out
	default:
		return nil
	}
}

func (s Snapshot) DeviceID() string   { return s.String(KeyDeviceID) }
func (s Snapshot) DeviceName() string { return s.String(KeyDeviceName) }

func (s Snapshot) Equal(other Snapshot) bool {
	if len(s) != len(other) {
		return false
	}
	for k, v := range s {
		ov, ok := other[k]
		if !ok || !valuesEqual(v, ov) {
			return false
		}
	}
	return true
}

func (s Snapshot) EqualExcept(other Snapshot, ignore map[string]bool) bool {
	for k, v := range s {
		if ignore[k] {
			continue
		}
		ov, ok := other[k]
		if !ok || !valuesEqual(v, ov) {
			return false
		}
	}
	for k := range other {
		if ignore[k] {
			continue
		}
		if _, ok := s[k]; !ok {
			return false
		}
	}
	return true
}

func valuesEqual(a, b any) bool {
	if af, ok := toFloat(a); ok {
		if bf, ok := toFloat(b); ok {
			return af == bf
		}
		return false
	}
	return reflect.DeepEqual(a, b)
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}
