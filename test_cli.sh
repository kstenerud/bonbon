#!/bin/bash
# ABOUTME: Command-line integration tests for bonbon

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
go build -o bonbon || { echo "Build failed"; exit 1; }

# Test: j2b command - convert JSON to BONJSON
echo '{"hello": "world"}' | ./bonbon j2b - "$TMPDIR/test.boj"
if [ -f "$TMPDIR/test.boj" ]; then
    pass "j2b: JSON to BONJSON conversion creates output file"
else
    fail "j2b: JSON to BONJSON conversion creates output file"
fi

# Test: b2j command - BONJSON to JSON conversion
./bonbon b2j "$TMPDIR/test.boj" - > "$TMPDIR/output.json"
if grep -q '"hello"' "$TMPDIR/output.json"; then
    pass "b2j: BONJSON to JSON conversion produces valid JSON"
else
    fail "b2j: BONJSON to JSON conversion produces valid JSON"
fi

# Test: Round-trip preserves data
echo '{"test": [1, 2, 3]}' > "$TMPDIR/input.json"
./bonbon j2b "$TMPDIR/input.json" "$TMPDIR/round.boj"
./bonbon b2j "$TMPDIR/round.boj" "$TMPDIR/round.json"
if grep -q '"test"' "$TMPDIR/round.json" && grep -q '1' "$TMPDIR/round.json"; then
    pass "Round-trip preserves data"
else
    fail "Round-trip preserves data"
fi

# Test: j command - validate JSON (valid)
if echo '{"valid": true}' | ./bonbon j - 2>/dev/null; then
    pass "j: validates valid JSON"
else
    fail "j: validates valid JSON"
fi

# Test: j command - validate JSON (invalid)
if echo '{"invalid": }' | ./bonbon j - 2>/dev/null; then
    fail "j: rejects invalid JSON"
else
    pass "j: rejects invalid JSON"
fi

# Test: b command - validate BONJSON (valid)
if ./bonbon b "$TMPDIR/test.boj" 2>/dev/null; then
    pass "b: validates valid BONJSON"
else
    fail "b: validates valid BONJSON"
fi

# Test: b command - validate BONJSON (invalid)
printf '\x9a\x81\x61' > "$TMPDIR/invalid.boj"  # Truncated object
if ./bonbon b "$TMPDIR/invalid.boj" 2>/dev/null; then
    fail "b: rejects invalid BONJSON"
else
    pass "b: rejects invalid BONJSON"
fi

# Test: j2j command - JSON to JSON (reformat)
echo '{"a":1,"b":2}' | ./bonbon j2j - "$TMPDIR/reformatted.json"
if grep -q '"a"' "$TMPDIR/reformatted.json" && grep -q '    ' "$TMPDIR/reformatted.json"; then
    pass "j2j: reformats JSON with indentation"
else
    fail "j2j: reformats JSON with indentation"
fi

# Test: b2b command - BONJSON to BONJSON (reformat)
./bonbon b2b "$TMPDIR/test.boj" "$TMPDIR/reformatted.boj"
./bonbon b2j "$TMPDIR/reformatted.boj" - | grep -q '"hello"'
if [ $? -eq 0 ]; then
    pass "b2b: BONJSON round-trip works"
else
    fail "b2b: BONJSON round-trip works"
fi

# Test: -e option prints end offset
OFFSET=$(./bonbon -e b "$TMPDIR/test.boj" 2>&1 >/dev/null)
if [ -n "$OFFSET" ] && [ "$OFFSET" -gt 0 ] 2>/dev/null; then
    pass "-e option prints end offset"
else
    fail "-e option prints end offset (got: $OFFSET)"
fi

# Test: -s option skips bytes
printf 'HEADER' > "$TMPDIR/header.boj"
cat "$TMPDIR/test.boj" >> "$TMPDIR/header.boj"
if ./bonbon -s 6 b2j "$TMPDIR/header.boj" - | grep -q '"hello"'; then
    pass "-s option skips bytes correctly"
else
    fail "-s option skips bytes correctly"
fi

# Test: -t option allows trailing data
printf '\x01garbage' > "$TMPDIR/trailing.boj"
if ./bonbon -t b2j "$TMPDIR/trailing.boj" - 2>/dev/null | grep -q '1'; then
    pass "-t option allows trailing data"
else
    fail "-t option allows trailing data"
fi

# Test: Trailing data without -t produces error but still outputs
OUTPUT=$(./bonbon b2j "$TMPDIR/trailing.boj" - 2>/dev/null || true)
EXITCODE=$(./bonbon b2j "$TMPDIR/trailing.boj" - >/dev/null 2>&1; echo $?)
if [ "$OUTPUT" = "1" ] && [ "$EXITCODE" != "0" ]; then
    pass "Trailing data without -t outputs value and returns error"
else
    fail "Trailing data without -t outputs value and returns error (output: $OUTPUT, exit: $EXITCODE)"
fi

# Test: Truncated BONJSON returns error
printf '\x9a\x81\x61\x01' > "$TMPDIR/truncated.boj"  # Incomplete object
EXITCODE=$(./bonbon b2j "$TMPDIR/truncated.boj" - >/dev/null 2>&1; echo $?)
if [ "$EXITCODE" != "0" ]; then
    pass "Truncated BONJSON returns error"
else
    fail "Truncated BONJSON returns error"
fi

# Test: Unknown command produces error
if ./bonbon unknown - - 2>/dev/null; then
    fail "Unknown command produces error"
else
    pass "Unknown command produces error"
fi

# Test: Missing output file for conversion command produces error
if ./bonbon j2b - 2>/dev/null; then
    fail "Missing output file produces error"
else
    pass "Missing output file produces error"
fi

# Test: j command rejects output file argument
if echo '{}' | ./bonbon j - /tmp/out 2>/dev/null; then
    fail "j: rejects output file argument"
else
    pass "j: rejects output file argument"
fi

# Test: b command rejects output file argument
if ./bonbon b "$TMPDIR/test.boj" /tmp/out 2>/dev/null; then
    fail "b: rejects output file argument"
else
    pass "b: rejects output file argument"
fi

# Summary
echo ""
echo "Results: $PASS passed, $FAIL failed"
if [ "$FAIL" -gt 0 ]; then
    exit 1
fi
