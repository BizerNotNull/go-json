#!/usr/bin/env bash
# Steady-state json.Marshal benchmark. See autoresearch/marshal/main.go for workload and metric units.
set -euo pipefail
cd "$(dirname "$0")"

export GOMAXPROCS=1
export GOGC=100
export GOFLAGS=""
# Never download a toolchain or dependencies during measurement.
export GOTOOLCHAIN=local
export GOPROXY=off
export GOSUMDB=off

go_bin=""
if command -v go >/dev/null 2>&1; then
	go_bin="$(command -v go)"
elif command -v go.exe >/dev/null 2>&1; then
	go_bin="$(command -v go.exe)"
elif [[ -x /usr/local/go/bin/go ]]; then
	go_bin=/usr/local/go/bin/go
fi
if [[ -z "$go_bin" ]]; then
	echo "autoresearch: go not found" >&2
	exit 1
fi
exec "$go_bin" run ./autoresearch/marshal
