#!/usr/bin/env bash
# Loopback smoke test (CX-TEST-001): build connex, start the agent, and exercise
# ls / run / exit-code passthrough / local fallback / down on a single machine.
# Uses an isolated CONNEX_HOME so it never touches your real key or agent.
set -euo pipefail
cd "$(dirname "$0")/.."

export CONNEX_HOME="$(pwd)/.connex-smoke"
rm -rf "$CONNEX_HOME"
trap 'rm -rf "$CONNEX_HOME"; rm -f ./connex-smoke-bin' EXIT

echo "==> build"
CGO_ENABLED=0 go build -o ./connex-smoke-bin ./cmd/connex
bin=./connex-smoke-bin

echo "==> up -d"
"$bin" up -d
sleep 1

echo "==> ls"
"$bin" ls

echo "==> run --on self -- cat (stdin round-trip)"
got=$(echo "hello-from-connex" | "$bin" run --on "$(hostname -s)" -- cat)
[ "$got" = "hello-from-connex" ] || { echo "FAIL: stdin round-trip got '$got'"; exit 1; }

echo "==> exit-code passthrough (expect 7)"
set +e
"$bin" run --on "$(hostname -s)" -- sh -c "exit 7"
code=$?
set -e
[ "$code" -eq 7 ] || { echo "FAIL: exit code was $code, want 7"; exit 1; }

echo "==> local fallback (no --on)"
out=$("$bin" run -- echo local-ok)
[ "$out" = "local-ok" ] || { echo "FAIL: local fallback got '$out'"; exit 1; }

echo "==> down"
"$bin" down

echo "PASS: loopback smoke test"
