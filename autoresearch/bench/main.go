// Benchmark harness for github.com/goccy/go-json Unmarshal.
// Steady-state only: type decoders are compiled before timing.
// unmarshal_ns is one balanced mix operation. Repeat counts keep each
// workload near the same share of that number.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"

	gojson "github.com/goccy/go-json"
)

const (
	bulkN = 48
	numsN = 256

	// Repeats inside one mix operation. Chosen so each workload is
	// roughly the same fraction of unmarshal_ns.
	smallRepeat  = 40
	wideRepeat   = 20
	nestedRepeat = 10
	numsRepeat   = 3
	anyRepeat    = 4

	chunkCount = 9
	allocRuns  = 200
)

type benchCase struct {
	name  string
	fn    func()
	iters int
}

type smallPayload struct {
	St   int    `json:"st"`
	Sid  int    `json:"sid"`
	Tt   string `json:"tt"`
	Gr   int    `json:"gr"`
	UUID string `json:"uuid"`
	IP   string `json:"ip"`
	UA   string `json:"ua"`
	TZ   int    `json:"tz"`
	V    int    `json:"v"`
}

type widePayload struct {
	A int     `json:"a"`
	B int     `json:"b"`
	C int64   `json:"c"`
	D int64   `json:"d"`
	E string  `json:"e"`
	F string  `json:"f"`
	G bool    `json:"g"`
	H bool    `json:"h"`
	I *int    `json:"i"`
	J *string `json:"j"`
	K float64 `json:"k"`
	L float64 `json:"l"`
	M string  `json:"m"`
	N string  `json:"n"`
	O int     `json:"o"`
	P int     `json:"p"`
	Q string  `json:"q"`
	R string  `json:"r"`
	S bool    `json:"s"`
	T int     `json:"t"`
	U string  `json:"u"`
	V int     `json:"v"`
	W string  `json:"w"`
	X int     `json:"x"`
}

type item struct {
	SKU   string  `json:"sku"`
	Qty   int     `json:"qty"`
	Price float64 `json:"price"`
	Note  *string `json:"note"`
}

type nestedPayload struct {
	ID     string         `json:"id"`
	Active bool           `json:"active"`
	Name   string         `json:"name"`
	Email  string         `json:"email"`
	Score  float64        `json:"score"`
	Tags   []string       `json:"tags"`
	Extra  map[string]int `json:"extra"`
	Items  []item         `json:"items"`
	Nick   *string        `json:"nick"`
	Code   code           `json:"code"`
}

// code exercises the Unmarshaler call path. Fixture values need no escapes.
type code string

func (c *code) UnmarshalJSON(b []byte) error {
	if len(b) < 2 || b[0] != '"' || b[len(b)-1] != '"' {
		return fmt.Errorf("code: %s", b)
	}
	*c = code(b[1 : len(b)-1])
	return nil
}

var (
	smallJSON  []byte
	wideJSON   []byte
	nestedJSON []byte
	bulkJSON   []byte
	numsJSON   []byte
	keep       int64
)

func main() {
	runtime.LockOSThread()
	raisePriority()
	buildFixtures()
	if err := checkFixtures(); err != nil {
		fatal(err)
	}
	warmup()

	cases := []benchCase{
		{name: "mix", fn: oneMix, iters: 1200},
		{name: "small", fn: oneSmall, iters: 400000},
		{name: "wide", fn: oneWide, iters: 160000},
		{name: "nested", fn: oneNested, iters: 50000},
		{name: "bulk", fn: oneBulk, iters: 6000},
		{name: "nums", fn: oneNums, iters: 25000},
		{name: "any", fn: oneAny, iters: 40000},
	}
	timed := make([]int64, len(cases))
	spread := make([]int64, len(cases))
	for i, c := range cases {
		timed[i], spread[i] = sampleNs(c.fn, c.iters)
	}
	if keep == 0 {
		fatal(fmt.Errorf("sink collapsed"))
	}
	for i, c := range cases {
		allocs, bytes := countAllocs(c.fn)
		fmt.Fprintf(os.Stderr, "%s ns=%d spread=%d allocs=%d bytes=%d\n", c.name, timed[i], spread[i], allocs, bytes)
		if c.name == "mix" {
			fmt.Printf("METRIC unmarshal_ns=%d\n", timed[i])
			fmt.Printf("METRIC allocs=%d\n", allocs)
			fmt.Printf("METRIC alloc_bytes=%d\n", bytes)
			continue
		}
		fmt.Printf("METRIC %s_ns=%d\n", c.name, timed[i])
		fmt.Printf("METRIC %s_allocs=%d\n", c.name, allocs)
		fmt.Printf("METRIC %s_alloc_bytes=%d\n", c.name, bytes)
	}
}

func oneMix() {
	for range smallRepeat {
		oneSmall()
	}
	for range wideRepeat {
		oneWide()
	}
	for range nestedRepeat {
		oneNested()
	}
	oneBulk()
	for range numsRepeat {
		oneNums()
	}
	for range anyRepeat {
		oneAny()
	}
}

func oneSmall() {
	var dst smallPayload
	if err := gojson.Unmarshal(smallJSON, &dst); err != nil {
		fatal(err)
	}
	keep += int64(dst.St + dst.Sid + dst.V)
}

func oneWide() {
	var dst widePayload
	if err := gojson.Unmarshal(wideJSON, &dst); err != nil {
		fatal(err)
	}
	keep += int64(dst.A + dst.B + dst.T)
	if dst.J != nil {
		keep += int64(len(*dst.J))
	}
}

func oneNested() {
	var dst nestedPayload
	if err := gojson.Unmarshal(nestedJSON, &dst); err != nil {
		fatal(err)
	}
	keep += int64(len(dst.Items) + len(dst.Tags) + len(dst.Code))
}

func oneBulk() {
	var dst []smallPayload
	if err := gojson.Unmarshal(bulkJSON, &dst); err != nil {
		fatal(err)
	}
	if len(dst) != bulkN {
		fatal(fmt.Errorf("bulk len %d", len(dst)))
	}
	keep += int64(dst[0].St + dst[len(dst)-1].Sid)
}

func oneNums() {
	var dst []int64
	if err := gojson.Unmarshal(numsJSON, &dst); err != nil {
		fatal(err)
	}
	if len(dst) != numsN {
		fatal(fmt.Errorf("nums len %d", len(dst)))
	}
	keep += dst[0] + dst[len(dst)-1]
}

func oneAny() {
	var dst any
	if err := gojson.Unmarshal(nestedJSON, &dst); err != nil {
		fatal(err)
	}
	m, ok := dst.(map[string]any)
	if !ok {
		fatal(fmt.Errorf("any type %T", dst))
	}
	keep += int64(len(m))
}

// sampleNs times fn for a fixed iteration count. The heap is reset before
// each chunk so allocator cost does not drift. The first chunk is dropped.
// The reported value is the second-lowest remaining chunk.
func sampleNs(fn func(), iters int) (int64, int64) {
	samples := make([]int64, chunkCount)
	for i := range chunkCount {
		d := timeLoop(iters, fn)
		samples[i] = d.Nanoseconds() / int64(iters)
		if samples[i] <= 0 {
			fatal(fmt.Errorf("non-positive ns/op"))
		}
	}
	samples = samples[1:]
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	return samples[1], samples[len(samples)-1] - samples[0]
}

func timeLoop(n int, fn func()) time.Duration {
	runtime.GC()
	boost(20 * time.Millisecond)
	prev := debug.SetGCPercent(-1)
	start := time.Now()
	for range n {
		fn()
	}
	elapsed := time.Since(start)
	debug.SetGCPercent(prev)
	return elapsed
}

func boost(d time.Duration) {
	end := time.Now().Add(d)
	var x uint64 = 1
	for time.Now().Before(end) {
		for range 128 {
			x ^= x << 13
			x ^= x >> 7
			x ^= x << 17
		}
	}
	if x == 0 {
		fatal(fmt.Errorf("boost collapsed"))
	}
}

func countAllocs(fn func()) (int64, int64) {
	for range 20 {
		fn()
	}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for range allocRuns {
		fn()
	}
	runtime.ReadMemStats(&after)
	return int64(after.Mallocs-before.Mallocs) / allocRuns, int64(after.TotalAlloc-before.TotalAlloc) / allocRuns
}

func warmup() {
	for range 8 {
		oneMix()
	}
	keep = 0
}

func buildFixtures() {
	smallJSON = []byte(`{"st":1,"sid":486,"tt":"active","gr":0,"uuid":"de305d54-75b4-431b-adb2-eb6b9e546014","ip":"127.0.0.1","ua":"user_agent","tz":-6,"v":1}`)
	wideJSON = []byte(`{"a":1,"b":2,"c":3,"d":-4,"e":"alpha","f":"beta","g":true,"h":false,"i":9,"j":"jay","k":1.5,"l":-2.25,"m":"em","n":"en","o":15,"p":16,"q":"queue","r":"ar","s":true,"t":20,"u":"you","v":22,"w":"double-u","x":24,"unknown":true}`)
	nestedJSON = []byte(`{"id":"d50887ca-a6ce-4e59-b89f-14f0b5d03b03","active":true,"name":"caf\u00e9 \"json\"","email":"ada@example.com","score":12.5,"tags":["a","b","c"],"extra":{"z":1,"a":2},"items":[{"sku":"p0","qty":1,"price":1.25,"note":"n0"},{"sku":"p1","qty":2,"price":2.5,"note":null},{"sku":"p2","qty":3,"price":3.75,"note":"n2"}],"nick":null,"code":"AB-12","skipped":{"x":1}}`)

	var bulk strings.Builder
	bulk.Grow(bulkN * 96)
	bulk.WriteByte('[')
	for i := range bulkN {
		if i > 0 {
			bulk.WriteByte(',')
		}
		bulk.WriteString(`{"st":`)
		bulk.WriteString(strconv.Itoa(i))
		bulk.WriteString(`,"sid":`)
		bulk.WriteString(strconv.Itoa(1000 + i))
		bulk.WriteString(`,"tt":"active","gr":0,"uuid":"id-`)
		bulk.WriteString(strconv.Itoa(i))
		bulk.WriteString(`","ip":"10.0.0.`)
		bulk.WriteString(strconv.Itoa(i % 256))
		bulk.WriteString(`","ua":"ua","tz":-6,"v":1}`)
	}
	bulk.WriteByte(']')
	bulkJSON = []byte(bulk.String())

	var nums strings.Builder
	nums.Grow(numsN * 8)
	nums.WriteByte('[')
	for i := range numsN {
		if i > 0 {
			nums.WriteByte(',')
		}
		nums.WriteString(strconv.Itoa(i*3 - 17))
	}
	nums.WriteByte(']')
	numsJSON = []byte(nums.String())
}

func checkFixtures() error {
	var gotS, wantS smallPayload
	if err := decodeBoth(smallJSON, &gotS, &wantS); err != nil {
		return err
	}
	if gotS.St != 1 || gotS.Sid != 486 || gotS.UUID == "" || gotS.TZ != -6 {
		return fmt.Errorf("small fields: %+v", gotS)
	}
	var gotW, wantW widePayload
	if err := decodeBoth(wideJSON, &gotW, &wantW); err != nil {
		return err
	}
	if gotW.I == nil || *gotW.I != 9 || gotW.J == nil || *gotW.J != "jay" || gotW.D != -4 {
		return fmt.Errorf("wide fields: %+v", gotW)
	}
	var gotN, wantN nestedPayload
	if err := decodeBoth(nestedJSON, &gotN, &wantN); err != nil {
		return err
	}
	if gotN.Name != "café \"json\"" || gotN.Code != "AB-12" || gotN.Nick != nil || len(gotN.Items) != 3 || gotN.Items[1].Note != nil {
		return fmt.Errorf("nested fields: %+v", gotN)
	}
	if gotN.Extra["a"] != 2 || gotN.Extra["z"] != 1 {
		return fmt.Errorf("nested map: %+v", gotN.Extra)
	}
	var gotB, wantB []smallPayload
	if err := decodeBoth(bulkJSON, &gotB, &wantB); err != nil {
		return err
	}
	if len(gotB) != bulkN || gotB[bulkN-1].Sid != 1000+bulkN-1 {
		return fmt.Errorf("bulk: len %d", len(gotB))
	}
	var gotNums, wantNums []int64
	if err := decodeBoth(numsJSON, &gotNums, &wantNums); err != nil {
		return err
	}
	if len(gotNums) != numsN || gotNums[1] != 3-17 {
		return fmt.Errorf("nums: %+v", gotNums[:2])
	}
	var gotAny, wantAny any
	if err := decodeBoth(nestedJSON, &gotAny, &wantAny); err != nil {
		return err
	}
	return nil
}

func decodeBoth(data []byte, got, want any) error {
	if err := gojson.Unmarshal(data, got); err != nil {
		return fmt.Errorf("go-json: %w", err)
	}
	if err := json.Unmarshal(data, want); err != nil {
		return fmt.Errorf("encoding/json: %w", err)
	}
	if !reflect.DeepEqual(got, want) {
		return fmt.Errorf("mismatch\n got %#v\nwant %#v", got, want)
	}
	return nil
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "autoresearch: %v\n", err)
	os.Exit(1)
}
