#!/usr/bin/env bash
# Loopback smoke test (KN-TEST-001): build knit, start the agent, and exercise
# ls / run / exit-code passthrough / local fallback / down on a single machine.
# Uses an isolated KNIT_HOME so it never touches your real key or agent.
set -euo pipefail
cd "$(dirname "$0")/.."

export KNIT_HOME="$(pwd)/.knit-smoke"
rm -rf "$KNIT_HOME"
trap 'rm -rf "$KNIT_HOME"; rm -f ./knit-smoke-bin' EXIT

echo "==> build"
CGO_ENABLED=0 go build -o ./knit-smoke-bin ./cmd/knit
bin=./knit-smoke-bin

echo "==> up -d"
"$bin" up -d
sleep 1

echo "==> gauge"
"$bin" gauge

echo "==> run --on self -- cat (stdin round-trip)"
got=$(echo "hello-from-knit" | "$bin" run --on "$(hostname -s)" -- cat)
[ "$got" = "hello-from-knit" ] || { echo "FAIL: stdin round-trip got '$got'"; exit 1; }

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
