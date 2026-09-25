package cardlayout

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"statusphere-client/internal/presence"
)

type field struct {
	key     string
	icon    string
	label   string
	value   string
	note    string
	percent *float64
}

var nativeFieldKeys = []string{"cpu", "mem", "ram", "memory", "disk", "load", "uptime", "workspace", "active_app", "active_window", "package_count"}

var iconsByKey = map[string]string{
	"cpu":        "planner_review",
	"mem":        "memory",
	"ram":        "memory",
	"memory":     "memory",
	"disk":       "storage",
	"gpu":        "deployed_code",
	"project":    "terminal",
	"workspace":  "desktop_windows",
	"language":   "code",
	"mood":       "mood",
	"region":     "location_on",
	"top_artist": "album",
	"genre":      "library_music",
}

var percentPattern = regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*%$`)

func fieldsFor(d presence.Snapshot) []field {
	if d == nil {
		return nil
	}
	fields := systemFieldsFor(d)
	if ws := d[presence.KeyActiveWorkspace]; truthy(ws) {
		fields = append(fields, field{key: "workspace", icon: "desktop_windows", label: "Workspace", value: jsString(ws)})
	}
	if w := d.String(presence.KeyActiveWindow); w != "" {
		fields = append(fields, field{key: "active_window", icon: "web_asset", label: "Window", value: w})
	}
	if a := d.String(presence.KeyActiveApp); a != "" {
		fields = append(fields, field{key: "active_app", icon: "apps", label: "App", value: a})
	}
	if d.Has(presence.KeyPackageCount) {
		fields = append(fields, field{key: "package_count", icon: "inventory_2", label: "Packages", value: jsString(d[presence.KeyPackageCount])})
	}
	for _, key := range d.Strings(presence.KeyCustomFields) {
		if !customValuePresent(d[key]) || slices.Contains(nativeFieldKeys, key) {
			continue
		}
		raw := jsString(d[key])
		icon := d.String(key + "_icon")
		if icon == "" {
			icon = iconsByKey[key]
		}
		fields = append(fields, field{key: key, icon: icon, label: labelForKey(key), value: raw, percent: percentForField(key, raw, d)})
	}
	return fields
}

func systemFieldsFor(d presence.Snapshot) []field {
	var fields []field
	if cpu, ok := d.Float(presence.KeyCPUPercent); ok {
		fields = append(fields, field{key: "cpu", icon: "planner_review", label: "CPU", value: jsInt(cpu) + "%", percent: &cpu})
	}
	used, _ := d.Float(presence.KeyMemUsedMB)
	if total, _ := d.Float(presence.KeyMemTotalMB); total > 0 {
		percent := used / total * 100
		value := toFixed(used/1024, 1) + "/" + toFixed(total/1024, 1) + "G"
		fields = append(fields, field{key: "mem", icon: "memory", label: "Memory", value: value, percent: &percent})
	}
	if disk, ok := d.Float(presence.KeyDiskUsedPercent); ok {
		note := ""
		if free, ok := d.Float(presence.KeyDiskFreeGB); ok {
			note = jsInt(free) + "G free"
		}
		fields = append(fields, field{key: "disk", icon: "storage", label: "Disk", note: note, value: jsInt(disk) + "%", percent: &disk})
	}
	if battery, ok := d.Float(presence.KeyBatteryPercent); ok {
		label := "Battery"
		if charging, _ := d[presence.KeyBatteryCharging].(bool); charging {
			label = "Charging"
		}
		fields = append(fields, field{key: "battery", icon: "battery_full", label: label, value: jsInt(battery) + "%", percent: &battery})
	}
	if load, ok := d.Float(presence.KeyLoadAvg1m); ok {
		value := toFixed(load, 2)
		if cpus, _ := d.Float(presence.KeyCPUCount); cpus > 0 {
			value += " / " + jsNumber(cpus)
		}
		fields = append(fields, field{key: "load", icon: "speed", label: "Load", value: value})
	}
	if hours, ok := d.Float(presence.KeyUptimeHours); ok {
		fields = append(fields, field{key: "uptime", icon: "schedule", label: "Uptime", value: formatUptime(hours)})
	}
	return fields
}

func formatUptime(hours float64) string {
	switch {
	case hours < 1:
		return jsInt(hours*60) + "m"
	case hours < 48:
		return jsInt(hours) + "h"
	default:
		return jsInt(hours/24) + "d"
	}
}

func percentForField(key, raw string, d presence.Snapshot) *float64 {
	if m := percentPattern.FindStringSubmatch(raw); m != nil {
		p, _ := strconv.ParseFloat(m[1], 64)
		return &p
	}
	if key == "mem" || key == "ram" || key == "memory" {
		used, _ := d.Float(presence.KeyMemUsedMB)
		if total, _ := d.Float(presence.KeyMemTotalMB); total > 0 {
			p := used / total * 100
			return &p
		}
	}
	return nil
}

func labelForKey(key string) string {
	words := strings.Split(key, "_")
	for i, w := range words {
		if r, size := utf8.DecodeRuneInString(w); size > 0 {
			words[i] = string(unicode.ToUpper(r)) + w[size:]
		}
	}
	return strings.Join(words, " ")
}

// detailFieldsFor drops active_app/active_window: the row's status line already
// carries them, so the fallback detail grid does not repeat them as a lone tile.
func detailFieldsFor(d presence.Snapshot) []field {
	fields := fieldsFor(d)
	out := fields[:0]
	for _, f := range fields {
		if f.key != "active_app" && f.key != "active_window" {
			out = append(out, f)
		}
	}
	return out
}

func fieldFor(d presence.Snapshot, key string) *field {
	for _, f := range fieldsFor(d) {
		if f.key == key {
			return &f
		}
	}
	return nil
}

// customValuePresent differs from CardLayouts' falsy check on purpose: a numeric 0 is a value.
func customValuePresent(v any) bool {
	switch v := v.(type) {
	case nil:
		return false
	case string:
		return v != ""
	case bool:
		return v
	default:
		return true
	}
}

func truthy(v any) bool {
	switch v := v.(type) {
	case nil:
		return false
	case string:
		return v != ""
	case bool:
		return v
	case float64:
		return v != 0 && !math.IsNaN(v)
	case int:
		return v != 0
	case int64:
		return v != 0
	case json.Number:
		f, err := v.Float64()
		return err == nil && f != 0
	default:
		return true
	}
}

func jsString(v any) string {
	switch v := v.(type) {
	case nil:
		return ""
	case string:
		return v
	case float64:
		return jsNumber(v)
	case json.Number:
		return v.String()
	default:
		return fmt.Sprint(v)
	}
}

func jsNumber(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func jsInt(f float64) string {
	return jsNumber(math.Round(f))
}

// toFixed rounds halves away from zero like Number.prototype.toFixed, where strconv rounds them to even.
func toFixed(f float64, digits int) string {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return jsNumber(f)
	}
	return new(big.Rat).SetFloat64(f).FloatString(digits)
}
