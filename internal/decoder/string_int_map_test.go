package decoder_test

import (
	"encoding/json"
	"reflect"
	"strconv"
	"testing"

	gojson "github.com/goccy/go-json"
)

func TestStringIntMapBoundaries(t *testing.T) {
	for _, input := range []string{
		`{}`, `null`, `{"a":1,"a":2}`, `{"a":null}`, `{"a":1,"b":null}`,
		`{"a" : -2, "b":3}`, `{"a\u0062":4}`, `{"a":1.5,"b":3}`,
		`{"a":9223372036854775807,"b":-9223372036854775808}`,
		`{"a":9223372036854775808}`, `{"a":1,"b":9223372036854775808}`,
		`{"a":01}`, `{"a":1,}`, `{"a":1,"b":2,"c":3,"d":4,"e":5,"f":6,"g":7,"h":8,"i":9}`,
	} {
		t.Run(input, func(t *testing.T) {
			got := map[string]int{"keep": 7, "a": 9}
			want := map[string]int{"keep": 7, "a": 9}
			gotErr := gojson.Unmarshal([]byte(input), &got)
			wantErr := json.Unmarshal([]byte(input), &want)
			if (gotErr == nil) != (wantErr == nil) {
				t.Fatalf("errors: got %v, want %v (%d-bit int)", gotErr, wantErr, strconv.IntSize)
			}
			// State following malformed JSON is not part of the API contract.
			if json.Valid([]byte(input)) && !reflect.DeepEqual(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
		})
	}
}
