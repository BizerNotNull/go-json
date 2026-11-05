package json_test

import (
	stdjson "encoding/json"
	"math"
	"regexp"
	"testing"

	"github.com/goccy/go-json"
)

type sliceIntegerMarshaler int64

func (v sliceIntegerMarshaler) MarshalJSON() ([]byte, error) {
	return []byte(`"custom"`), nil
}

// Slice traversal must preserve element widths, delimiters, nesting and custom
// methods in every VM, including when the slice follows another encoded value.
func TestEncodeIntegerSlices(t *testing.T) {
	type namedInt int32
	var nilSlice []int64
	empty := []int64{}
	n := int64(-7)
	values := []any{
		nilSlice, &nilSlice, empty, &empty,
		[]int{0, -1, 1},
		[]int8{math.MinInt8, -1, 0, math.MaxInt8},
		[]int16{math.MinInt16, -1, 0, math.MaxInt16},
		[]int32{math.MinInt32, -1, 0, math.MaxInt32},
		[]int64{math.MinInt64, -1, 0, math.MaxInt64},
		[]namedInt{-10000, 0, 10000},
		[][]int64{nil, {}, {1}, {-1, 2}},
		[]*int64{nil, &n},
		[]sliceIntegerMarshaler{1, 2},
		map[string][]int64{"a": {1, -2}, "b": {}, "c": nil},
		struct {
			Before string
			Values []int64
			After  bool
		}{"before", []int64{-10, 20}, true},
	}
	colors := regexp.MustCompile("\x1b\\[[0-9;]*m")
	for _, v := range values {
		for _, indent := range []bool{false, true} {
			var want []byte
			var err error
			if indent {
				want, err = stdjson.MarshalIndent(v, "prefix:", "  ")
			} else {
				want, err = stdjson.Marshal(v)
			}
			if err != nil {
				t.Fatal(err)
			}
			for _, color := range []bool{false, true} {
				var opts []json.EncodeOptionFunc
				if color {
					opts = append(opts, json.Colorize(json.DefaultColorScheme))
				}
				var got []byte
				if indent {
					got, err = json.MarshalIndentWithOption(v, "prefix:", "  ", opts...)
				} else {
					got, err = json.MarshalWithOption(v, opts...)
				}
				if err != nil {
					t.Fatal(err)
				}
				if color {
					got = colors.ReplaceAll(got, nil)
				}
				if string(got) != string(want) {
					t.Errorf("%T indent=%v color=%v: got %s, want %s", v, indent, color, got, want)
				}
			}
		}
	}
}
