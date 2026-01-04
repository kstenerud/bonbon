#!/bin/bash
# ABOUTME: Command-line integration tests for j2b

set -e

PASS=0
FAIL=0
TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

pass() {
    echo "PASS: $1"
    PASS=$((PASS + 1))
}

fail() {
    echo "FAIL: $1"
    FAIL=$((FAIL + 1))
}

# Build first
go build -o j2b || { echo "Build failed"; exit 1; }

# Test: JSON to BONJSON conversion
echo '{"hello": "world"}' | ./j2b - "$TMPDIR/test.boj"
if [ -f "$TMPDIR/test.boj" ]; then
    pass "JSON to BONJSON conversion creates output file"
else
    fail "JSON to BONJSON conversion creates output file"
fi

# Test: BONJSON to JSON conversion
./j2b "$TMPDIR/test.boj" - > "$TMPDIR/output.json"
if grep -q '"hello"' "$TMPDIR/output.json"; then
    pass "BONJSON to JSON conversion produces valid JSON"
else
    fail "BONJSON to JSON conversion produces valid JSON"
fi

# Test: Round-trip preserves data
echo '{"test": [1, 2, 3]}' > "$TMPDIR/input.json"
./j2b "$TMPDIR/input.json" "$TMPDIR/round.boj"
./j2b "$TMPDIR/round.boj" "$TMPDIR/round.json"
if grep -q '"test"' "$TMPDIR/round.json" && grep -q '1' "$TMPDIR/round.json"; then
    pass "Round-trip preserves data"
else
    fail "Round-trip preserves data"
fi

# Test: Validate-only mode with valid JSON (no output file)
if echo '{"valid": true}' | ./j2b - 2>/dev/null; then
    pass "Validate-only mode accepts valid JSON"
else
    fail "Validate-only mode accepts valid JSON"
fi

# Test: Validate-only mode with invalid JSON
if echo '{"invalid": }' | ./j2b - 2>/dev/null; then
    fail "Validate-only mode rejects invalid JSON"
else
    pass "Validate-only mode rejects invalid JSON"
fi

# Test: Validate-only mode with valid BONJSON
if ./j2b "$TMPDIR/test.boj" 2>/dev/null; then
    pass "Validate-only mode accepts valid BONJSON"
else
    fail "Validate-only mode accepts valid BONJSON"
fi

# Test: -e option prints end offset
OFFSET=$(./j2b -e "$TMPDIR/test.boj" - 2>&1 >/dev/null)
if [ -n "$OFFSET" ] && [ "$OFFSET" -gt 0 ] 2>/dev/null; then
    pass "-e option prints end offset"
else
    fail "-e option prints end offset (got: $OFFSET)"
fi

# Test: -s option skips bytes
printf 'HEADER' > "$TMPDIR/header.boj"
cat "$TMPDIR/test.boj" >> "$TMPDIR/header.boj"
if ./j2b -s 6 "$TMPDIR/header.boj" - | grep -q '"hello"'; then
    pass "-s option skips bytes correctly"
else
    fail "-s option skips bytes correctly"
fi

# Test: -t option allows trailing data
printf '\x01garbage' > "$TMPDIR/trailing.boj"
if ./j2b -t "$TMPDIR/trailing.boj" - 2>/dev/null | grep -q '1'; then
    pass "-t option allows trailing data"
else
    fail "-t option allows trailing data"
fi

# Test: Trailing data without -t produces error but still outputs
OUTPUT=$(./j2b "$TMPDIR/trailing.boj" - 2>/dev/null || true)
EXITCODE=$(./j2b "$TMPDIR/trailing.boj" - >/dev/null 2>&1; echo $?)
if [ "$OUTPUT" = "1" ] && [ "$EXITCODE" != "0" ]; then
    pass "Trailing data without -t outputs value and returns error"
else
    fail "Trailing data without -t outputs value and returns error (output: $OUTPUT, exit: $EXITCODE)"
fi

# Test: Truncated BONJSON outputs partial result and returns error
printf '\x9a\x81\x61\x01' > "$TMPDIR/truncated.boj"  # Incomplete object
OUTPUT=$(./j2b "$TMPDIR/truncated.boj" - 2>/dev/null || true)
EXITCODE=$(./j2b "$TMPDIR/truncated.boj" - >/dev/null 2>&1; echo $?)
if [ "$EXITCODE" != "0" ]; then
    pass "Truncated BONJSON returns error"
else
    fail "Truncated BONJSON returns error"
fi

# Summary
echo ""
echo "Results: $PASS passed, $FAIL failed"
if [ "$FAIL" -gt 0 ]; then
    exit 1
fi
