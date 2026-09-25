package cardlayout

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"statusphere-client/internal/presence"
)

type roomCase struct {
	Members []presence.Snapshot `json:"members"`
	Photos  []struct {
		AccountID string `json:"account_id"`
		Path      string `json:"path"`
	} `json:"photos"`
}

func readJSON(t *testing.T, path string, into any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, into); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
}

func asJSONValue(t *testing.T, v any) any {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestCardsMatchWidget(t *testing.T) {
	var cases map[string]roomCase
	var golden map[string]json.RawMessage
	readJSON(t, "testdata/cases.json", &cases)
	readJSON(t, "testdata/golden.json", &golden)

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			photos := map[string]string{}
			for _, p := range c.Photos {
				photos[p.AccountID] = p.Path
			}
			want, ok := golden[name]
			if !ok {
				t.Fatalf("no golden output for %s, rerun testdata/golden.mjs", name)
			}
			var wantValue any
			if err := json.Unmarshal(want, &wantValue); err != nil {
				t.Fatal(err)
			}

			got := asJSONValue(t, Cards(c.Members, photos))

			if !reflect.DeepEqual(got, wantValue) {
				gotText, _ := json.MarshalIndent(got, "", "  ")
				t.Errorf("cards differ from the widget\ngot:\n%s\nwant:\n%s", gotText, want)
			}
		})
	}
}

func TestZeroCustomValueStaysOnCard(t *testing.T) {
	member := presence.Snapshot{
		presence.KeyDeviceID:     "d",
		presence.KeyCustomFields: []string{"unread"},
		"unread":                 0,
	}

	detail := Cards([]presence.Snapshot{member}, nil)[0].Detail

	if len(detail) != 1 || detail[0].Field != "unread" || detail[0].Value != "0" {
		t.Fatalf("detail = %+v, want one unread tile showing 0", detail)
	}
}
