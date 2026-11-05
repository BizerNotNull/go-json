// Command marshal measures the steady-state root Marshal API, including its
// caller-owned output allocation. Fixtures, repeat counts, and iteration counts
// are fixed; encoder compilation and correctness checks are outside timing.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	gojson "github.com/goccy/go-json"
)

const (
	sampleCount    = 7
	mixIterations  = 4000
	allocationRuns = 200
)

type record struct {
	ID       int64   `json:"id"`
	Name     string  `json:"name"`
	Active   bool    `json:"active"`
	Score    float64 `json:"score"`
	Optional string  `json:"optional,omitempty"`
}

type wideRecord struct {
	A, B, C, D int64
	E, F, G, H string
	I, J, K, L float64
	M, N       bool
	O          *int
	P          *string
	Q          string `json:",string"`
	R          int64  `json:",string"`
	S          []byte
	T          string `json:",omitempty"`
}

type envelope struct {
	Title   string            `json:"title"`
	Records []record          `json:"records"`
	Labels  map[string]string `json:"labels"`
	Next    *record           `json:"next"`
	Missing *record           `json:"missing,omitempty"`
}

type customValue int

func (v customValue) MarshalJSON() ([]byte, error) {
	return []byte(` { "status" : "ready", "html" : "<ok>&" } `), nil
}

type workload struct {
	name       string
	values     []any
	repeats    int
	iterations int
	want       [][]byte
}

var sink []byte

func main() {
	runtime.GOMAXPROCS(1)
	cases := fixtures()
	for i := range cases {
		c := &cases[i]
		for _, v := range c.values {
			want, err := json.Marshal(v)
			if err != nil {
				fail(err)
			}
			c.want = append(c.want, want)
		}
	}
	check(cases)
	mix := func() {
		for i := range cases {
			for range cases[i].repeats {
				run(&cases[i])
			}
		}
	}
	for range 100 {
		mix()
	}
	fmt.Fprintf(os.Stderr, "toolchain=%s os=%s arch=%s samples=%d mix_iterations=%d GOMAXPROCS=1 GOGC=100\n", runtime.Version(), runtime.GOOS, runtime.GOARCH, sampleCount, mixIterations)
	// Buffer all metrics until the post-timing correctness check succeeds.
	var metrics strings.Builder
	report(&metrics, "marshal", mix, mixIterations)
	for i := range cases {
		c := &cases[i]
		report(&metrics, c.name, func() { run(c) }, c.iterations)
	}
	check(cases)
	fmt.Print(metrics.String())
	runtime.KeepAlive(sink)
}

func run(c *workload) {
	for _, v := range c.values {
		b, err := gojson.Marshal(v)
		if err != nil {
			fail(fmt.Errorf("%s: %w", c.name, err))
		}
		sink = b
	}
}

func check(cases []workload) {
	for _, c := range cases {
		for i, v := range c.values {
			got, err := gojson.Marshal(v)
			if err != nil {
				fail(err)
			}
			if !bytes.Equal(got, c.want[i]) {
				fail(fmt.Errorf("%s[%d]: got %s, want %s", c.name, i, got, c.want[i]))
			}
			// A later call must not overwrite an earlier caller-owned result.
			for range 3 {
				run(&c)
			}
			if !bytes.Equal(got, c.want[i]) {
				fail(fmt.Errorf("%s[%d]: output aliased", c.name, i))
			}
		}
	}
}

func report(out *strings.Builder, name string, fn func(), iterations int) {
	samples := make([]float64, sampleCount)
	for i := range samples {
		runtime.GC()
		start := time.Now()
		for range iterations {
			fn()
		}
		samples[i] = float64(time.Since(start).Nanoseconds()) / float64(iterations)
		if samples[i] <= 0 {
			fail(fmt.Errorf("%s: non-positive timing", name))
		}
	}
	sort.Float64s(samples)
	for range 20 {
		fn()
	}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for range allocationRuns {
		fn()
	}
	runtime.ReadMemStats(&after)
	fmt.Fprintf(out, "METRIC %s_ns=%.3f\n", name, samples[sampleCount/2])
	fmt.Fprintf(out, "METRIC %s_allocs=%.3f\n", name, float64(after.Mallocs-before.Mallocs)/allocationRuns)
	fmt.Fprintf(out, "METRIC %s_alloc_bytes=%.3f\n", name, float64(after.TotalAlloc-before.TotalAlloc)/allocationRuns)
	fmt.Fprintf(os.Stderr, "%s: median %.3f ns, min %.3f, max %.3f (one fixed workload operation)\n", name, samples[sampleCount/2], samples[0], samples[len(samples)-1])
}

func fixtures() []workload {
	n, s := 42, "pointer value"
	basic := record{ID: 123456789, Name: "Ada Lovelace", Active: true, Score: 98.125}
	wide := wideRecord{A: -1, B: 123456789, C: 9223372036854775807, D: -9223372036854775808,
		E: "alpha", F: "beta", G: "gamma", H: "delta", I: 0.125, J: -123.75, K: 1e-12, L: 1e24,
		M: true, O: &n, P: &s, Q: "quoted", R: -12345, S: []byte("binary\x00payload")}
	bulk := make([]record, 48)
	for i := range bulk {
		bulk[i] = record{ID: int64(i*7919 - 1000), Name: fmt.Sprintf("account-%02d", i), Active: i%2 == 0, Score: float64(i) / 8}
	}
	nested := envelope{Title: "response", Records: bulk[:4], Labels: map[string]string{"z": "last", "a": "first", "region": "west"}, Next: &basic}
	nums := make([]int64, 256)
	for i := range nums {
		nums[i] = int64(i*i*7919 - 123456789)
	}
	text := []string{"plain ASCII", "caf\u00e9 \u4e2d\u6587", "quote\" slash\\ newline\n tab\t", "<script>&\u2028\u2029", "invalid\xffutf8", strings.Repeat("long unescaped text ", 32)}
	dynamic := map[string]any{"z": nil, "a": true, "count": int64(123456789), "ratio": 1.25, "nested": nested,
		"array": []any{"text", false, -42, map[string]any{"x": "<value>", "y": []int{1, 2, 3}}}}
	// Each operation visits all listed inputs. Primary mix weights are fixed;
	// secondary timings are per operation, not per individual Marshal call.
	return []workload{
		{name: "small", values: []any{&basic, basic}, repeats: 16, iterations: 60000},
		{name: "wide", values: []any{&wide}, repeats: 8, iterations: 30000},
		{name: "nested", values: []any{&nested}, repeats: 4, iterations: 12000},
		{name: "bulk", values: []any{bulk}, repeats: 1, iterations: 4000},
		{name: "nums", values: []any{nums}, repeats: 4, iterations: 10000},
		{name: "dynamic", values: []any{dynamic}, repeats: 2, iterations: 6000},
		{name: "escaped", values: []any{text}, repeats: 4, iterations: 16000},
		{name: "custom", values: []any{struct {
			Value customValue `json:"value"`
			Empty []int       `json:"empty"`
		}{Value: 7}}, repeats: 4, iterations: 30000},
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
