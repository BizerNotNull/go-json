# Repository Guidelines

## Project Overview

`github.com/goccy/go-json` is a performance-focused JSON library with an `encoding/json`-style root API, a `v2` API, and a streaming `jsontext` API. Read `README.md` for compatibility exceptions: behavior depends on the API, Go version, and operation. The library's `v2` and `jsontext` types are not standard-library aliases; migrate both imports together, including custom marshaler method signatures.

## Architecture & Data Flow

- Root entry points (`json.go`, `encode.go`, `decode.go`) and `v2/arshal*.go` establish options, obtain runtime contexts, dispatch work, and translate errors.
- Encoding: Go type → cached opcode program in `internal/encoder` → execution through `internal/encoder/run` and VM variants → output buffer or writer. Preserve compile-once/run-many behavior.
- Decoding: input and destination type → cached type decoder in `internal/decoder` → destination writes and error resolution. The v2 path reuses legacy decoders where semantics agree and has separate implementations where they differ.
- `jsontext` handles incremental token/value scanning, JSON grammar, container state, and streaming buffers; v2 integrates this layer with Go-value conversion.
- Shared type caches in `internal/runtime` use atomic publication. Runtime contexts are pooled; unsafe pointers, VM slots, buffer aliasing, and GC liveness are part of correctness.

## Key Directories

- `internal/encoder/`, `internal/decoder/`: type compilation, execution, caches, and API-specific semantics.
- `internal/runtime/`: runtime type/value access and shared cache machinery. Small sibling packages handle fields, strings, numbers, formatting, and options.
- `v2/`, `jsontext/`: distinct public API semantics; keep changes in the appropriate layer.
- `internal/cmd/`: generators and benchmark regression tooling.
- `benchmarks/`: separate Go module with comparison dependencies, fixtures, and report generation; root `./...` commands do not include it.
- `test/cover/`, `test/example/`: cross-package coverage cases and examples. `issues/README.md` indexes the maintainer work queue; check its status notes before selecting an item.

## Development Commands

Run from the repository root unless noted:

| Task | Command | Notes |
| --- | --- | --- |
| Build | `go build ./...` | Library and internal command packages; no application server to start. |
| Test | `go test ./...` | CI uses `go test -v ./... -count=1`. |
| Targeted test | `go test ./internal/decoder -run '^TestName$' -count=1` | Replace the package and test name. |
| Race check | `go test -race ./...` | Requires a supported race-detector platform/toolchain. |
| Coverage | `make cover` | Writes `cover.out`; `make cover-html` opens the report. |
| Lint | `make lint` | Installs pinned golangci-lint v2.14.0 into `bin/`, then runs it. |
| Generate | `make generate` | Runs `go generate ./internal/...`; review generated changes. |
| Performance gate | `make bench-check` | Compares go-json benchmarks against the base branch. Pass options with `BENCH_CHECK_FLAGS`. |

For isolated benchmarks, run `go test -run '^$' -bench '<pattern>' -benchmem .` from `benchmarks/`. Use `make bench-compare-encode`, `make bench-compare-decode`, or `make bench-report` for the documented comparison/report workflows. `make bench-compare-jsontext` needs Go 1.27; reports write under `benchmarks/site`.

## Code Conventions & Common Patterns

- Follow idiomatic Go naming, documented exported symbols, and the configured `gofmt`/`goimports` formatting. Keep options in the existing variadic option APIs.
- Public facades handle API contracts; internal packages own shared execution. Extend existing option/context mechanisms instead of adding a parallel configuration or dependency-injection layer.
- Preserve syntax-versus-semantic/type error distinctions, wrapped causes, offsets, and v2 JSON pointers. Root decoding checks trailing input and gives earlier syntax errors precedence over recorded type errors.
- Avoid per-value reflection, extra copying, and allocations in hot paths. Preserve cache publication ordering, pooled-context reset/release, pointer ownership, and `runtime.KeepAlive` requirements.
- Concurrency centers on shared caches and pools; streaming encoder/decoder state belongs to its instance. Use the existing synchronization patterns, not unsynchronized shared mutable state.
- Preserve Go-version, experiment, architecture, and race build constraints. Update alternate implementations when a shared contract changes.
- VM variants are generated: change `internal/cmd/generator/vm.go.tmpl` or the generator, then regenerate instead of patching only generated VM files. Numeric tables also have generation directives.

## Important Files

- `json.go`, `encode_of.go`, `decode_of.go`: root public API and typed allocation-sensitive entry points.
- `v2/arshal.go`, `v2/arshal_unmarshal.go`, `v2/errors.go`: v2 dispatch and structured semantic errors.
- `jsontext/encode.go`, `jsontext/decode.go`: streaming state and buffer ownership contracts.
- `internal/encoder/context.go`, `internal/runtime/type_cache.go`: VM slot liveness and cache synchronization constraints.
- `go.mod`, `benchmarks/go.mod`, `Makefile`, `.golangci.yml`, `.github/workflows/go.yml`: dependency, tooling, and CI contracts.

## Runtime/Tooling Preferences

- Root module: Go 1.23 minimum, no pinned `toolchain` directive or external module requirements. Use Go modules; there is no JavaScript package-manager workflow.
- Benchmark module: Go 1.24 minimum, its own dependencies, and a local replacement pointing to the root checkout. Some benchmark suites require newer Go versions.
- CI currently tests Go 1.25, 1.26, and 1.27.3 across Linux, macOS, and Windows, with additional 386, s390x, GC-pressure, and race checks.
- The library's v2 APIs do not require `GOEXPERIMENT=jsonv2`. Selected standard-library differential tests and implementations require Go 1.27 plus that experiment; check file build tags.
- Make targets use POSIX shell utilities. On Windows, use a compatible shell for Make, or direct Go commands where equivalent. Lint installation and benchmark dependencies may require network access.

## Testing & QA

- Use standard Go `testing`, colocated `*_test.go` files, `Test…`, `Benchmark…`, and `Fuzz…` naming. Follow nearby table-driven and standard-library comparison patterns.
- For behavior changes, add a focused regression test covering observable output, errors, streaming transitions, or ownership/lifetime behavior. Check both API layers when changing shared internals.
- Run targeted tests first, then `go test ./...`. For unsafe/cache changes, include race and applicable GC-pressure/platform checks; CI also exercises `GOGC=1`.
- Coverage configuration in `.octocov.yml` declares an 80% acceptable threshold and excludes generated VM directories. `make cover` excludes command and test-helper packages from its instrumentation list.
- Seeded fuzz cases run with ordinary tests; active fuzzing is separate. Example: `go test ./jsontext/internal/stdtest -run '^$' -fuzz '^FuzzCoder$' -fuzztime 10s`.
- `testdata/decode_as_go127.golden` records compatibility expectations. Its test's `-update` flag rewrites it and requires the documented Go 1.27/jsonv2 setup; do not refresh it to hide a regression.
- Check allocations and representative benchmarks for hot-path edits. Benchmark results and nested-module tests are not covered by the root test command.
