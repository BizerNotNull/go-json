#!/usr/bin/env bash
# Steady-state json.Unmarshal benchmark. Prints METRIC lines; exits non-zero on failure.
set -euo pipefail
cd "$(dirname "$0")"

export GOMAXPROCS=1
export GOGC=100
export GOFLAGS="${GOFLAGS:-}"
# This module has no external requirements. Fail instead of fetching.
export GOPROXY="${GOPROXY:-off}"
export GOSUMDB="${GOSUMDB:-off}"

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
exec "$go_bin" run ./autoresearch/bench
