package decoder_test

import (
	"encoding/json"
	"reflect"
	"testing"

	gojson "github.com/goccy/go-json"
)

func TestInt64SliceExistingStorage(t *testing.T) {
	for _, input := range []string{`[1,2]`, `[9223372036854775808,2]`, `[null,2]`} {
		t.Run(input, func(t *testing.T) {
			got := []int64{42}
			want := []int64{42}
			gotAlias, wantAlias := got, want
			gotErr := gojson.Unmarshal([]byte(input), &got)
			wantErr := json.Unmarshal([]byte(input), &want)
			if (gotErr == nil) != (wantErr == nil) || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(gotAlias, wantAlias) {
				t.Fatalf("got %v alias %v error %v; want %v alias %v error %v", got, gotAlias, gotErr, want, wantAlias, wantErr)
			}
		})
	}
}
