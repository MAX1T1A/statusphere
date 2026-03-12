package models

import "fmt"

type Snapshot map[string]any

func (s Snapshot) Equal(other Snapshot) bool {
	if len(s) != len(other) {
		return false
	}
	for k, v := range s {
		ov, ok := other[k]
		if !ok {
			return false
		}
		if fmt.Sprint(v) != fmt.Sprint(ov) {
			return false
		}
	}
	return true
}
