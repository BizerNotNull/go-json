package decoder_test

import (
	"encoding/json"
	"testing"

	gojson "github.com/goccy/go-json"
)

func TestFlatInt64SliceMatchesStdlib(t *testing.T) {
	inputs := []string{
		`[]`,
		`[0]`,
		`[1,2,3]`,
		`[-1,0,42]`,
		`[ 1 , -2 , 3 ]`,
		`[9223372036854775807,-9223372036854775808]`,
		`null`,
		`[1.5]`,
		`[1e2]`,
		`[01]`,
		`["x"]`,
		`[1,]`,
		`[1,2,3,]`,
		`{}`,
		`[1, [2]]`,
	}
	long := make([]byte, 0, 400)
	long = append(long, '[')
	for i := range 100 {
		if i > 0 {
			long = append(long, ',')
		}
		long = append(long, '1')
	}
	long = append(long, ']')
	inputs = append(inputs, string(long))

	for _, in := range inputs {
		var got, want []int64
		errGot := gojson.Unmarshal([]byte(in), &got)
		errWant := json.Unmarshal([]byte(in), &want)
		if (errGot == nil) != (errWant == nil) {
			t.Fatalf("input %s: go-json err %v, encoding/json err %v", in, errGot, errWant)
		}
		if errGot == nil && !sliceEqual(got, want) {
			t.Fatalf("input %s: got %v want %v", in, got, want)
		}
	}

	dst := make([]int64, 0, 8)
	if err := gojson.Unmarshal([]byte(`[4,5,6]`), &dst); err != nil {
		t.Fatal(err)
	}
	if cap(dst) < 8 || len(dst) != 3 || dst[0] != 4 || dst[2] != 6 {
		t.Fatalf("reuse: len %d cap %d %v", len(dst), cap(dst), dst)
	}
}

func sliceEqual(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
