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

# Test: -n option allows NUL characters
# Create BONJSON with NUL in string: 0x83 (string length 3) + "a\x00b"
printf '\x83a\x00b' > "$TMPDIR/nul.boj"
if ./bonbon b "$TMPDIR/nul.boj" 2>/dev/null; then
    fail "-n: NUL rejected by default"
else
    pass "-n: NUL rejected by default"
fi
if ./bonbon -n b "$TMPDIR/nul.boj" 2>/dev/null; then
    pass "-n: NUL allowed with -n flag"
else
    fail "-n: NUL allowed with -n flag"
fi

# Test: -d option for duplicate key handling
# Create BONJSON object with duplicate keys: {"a":1,"a":2}
# 0x9a = start object, 0x81 = string len 1, 'a', 0x01 = int 1, 0x81 'a' 0x02 = int 2, 0x9b = end object
printf '\x9a\x81a\x01\x81a\x02\x9b' > "$TMPDIR/dupkey.boj"
if ./bonbon b "$TMPDIR/dupkey.boj" 2>/dev/null; then
    fail "-d: duplicate keys rejected by default"
else
    pass "-d: duplicate keys rejected by default"
fi
OUTPUT=$(./bonbon -d keepfirst b2j "$TMPDIR/dupkey.boj" - 2>/dev/null)
if echo "$OUTPUT" | grep -q '"a": 1'; then
    pass "-d keepfirst: keeps first value"
else
    fail "-d keepfirst: keeps first value (got: $OUTPUT)"
fi
OUTPUT=$(./bonbon -d keeplast b2j "$TMPDIR/dupkey.boj" - 2>/dev/null)
if echo "$OUTPUT" | grep -q '"a": 2'; then
    pass "-d keeplast: keeps last value"
else
    fail "-d keeplast: keeps last value (got: $OUTPUT)"
fi

# Test: -u option for invalid UTF-8 handling
# Create BONJSON with invalid UTF-8: 0x83 (string len 3) + "a\xffb"
printf '\x83a\xffb' > "$TMPDIR/badutf8.boj"
if ./bonbon b "$TMPDIR/badutf8.boj" 2>/dev/null; then
    fail "-u: invalid UTF-8 rejected by default"
else
    pass "-u: invalid UTF-8 rejected by default"
fi
OUTPUT=$(./bonbon -u replace b2j "$TMPDIR/badutf8.boj" - 2>/dev/null)
if echo "$OUTPUT" | grep -q 'a.*b'; then
    pass "-u replace: replaces invalid bytes"
else
    fail "-u replace: replaces invalid bytes (got: $OUTPUT)"
fi
OUTPUT=$(./bonbon -u delete b2j "$TMPDIR/badutf8.boj" - 2>/dev/null)
if [ "$OUTPUT" = '"ab"' ]; then
    pass "-u delete: deletes invalid bytes"
else
    fail "-u delete: deletes invalid bytes (got: $OUTPUT)"
fi
if ./bonbon -u ignore b "$TMPDIR/badutf8.boj" 2>/dev/null; then
    pass "-u ignore: ignores invalid UTF-8"
else
    fail "-u ignore: ignores invalid UTF-8"
fi

# Test: invalid -d mode produces error
if ./bonbon -d invalid b - 2>/dev/null; then
    fail "-d: rejects invalid mode"
else
    pass "-d: rejects invalid mode"
fi

# Test: invalid -u mode produces error
if ./bonbon -u invalid b - 2>/dev/null; then
    fail "-u: rejects invalid mode"
else
    pass "-u: rejects invalid mode"
fi

# Test: -f option for special float (NaN/Infinity) handling
# Create BONJSON with NaN: 0x6c (64-bit float) + IEEE 754 NaN in little-endian
printf '\x6c\x01\x00\x00\x00\x00\x00\xf8\x7f' > "$TMPDIR/nan.boj"
# Create BONJSON with +Infinity
printf '\x6c\x00\x00\x00\x00\x00\x00\xf0\x7f' > "$TMPDIR/posinf.boj"
# Create BONJSON with -Infinity
printf '\x6c\x00\x00\x00\x00\x00\x00\xf0\xff' > "$TMPDIR/neginf.boj"

# Test: NaN rejected by default
if ./bonbon b "$TMPDIR/nan.boj" 2>/dev/null; then
    fail "-f: NaN rejected by default"
else
    pass "-f: NaN rejected by default"
fi

# Test: +Infinity rejected by default
if ./bonbon b "$TMPDIR/posinf.boj" 2>/dev/null; then
    fail "-f: +Infinity rejected by default"
else
    pass "-f: +Infinity rejected by default"
fi

# Test: -f allow mode allows NaN
if ./bonbon -f allow b "$TMPDIR/nan.boj" 2>/dev/null; then
    pass "-f allow: NaN allowed"
else
    fail "-f allow: NaN allowed"
fi

# Test: -f allow mode allows +Infinity
if ./bonbon -f allow b "$TMPDIR/posinf.boj" 2>/dev/null; then
    pass "-f allow: +Infinity allowed"
else
    fail "-f allow: +Infinity allowed"
fi

# Test: -f allow mode allows -Infinity
if ./bonbon -f allow b "$TMPDIR/neginf.boj" 2>/dev/null; then
    pass "-f allow: -Infinity allowed"
else
    fail "-f allow: -Infinity allowed"
fi

# Test: -f stringify mode converts NaN to string
OUTPUT=$(./bonbon -f stringify b2j "$TMPDIR/nan.boj" - 2>/dev/null)
if [ "$OUTPUT" = '"NaN"' ]; then
    pass "-f stringify: NaN becomes string"
else
    fail "-f stringify: NaN becomes string (got: $OUTPUT)"
fi

# Test: -f stringify mode converts +Infinity to string
OUTPUT=$(./bonbon -f stringify b2j "$TMPDIR/posinf.boj" - 2>/dev/null)
if [ "$OUTPUT" = '"Infinity"' ]; then
    pass "-f stringify: +Infinity becomes string"
else
    fail "-f stringify: +Infinity becomes string (got: $OUTPUT)"
fi

# Test: -f stringify mode converts -Infinity to string
OUTPUT=$(./bonbon -f stringify b2j "$TMPDIR/neginf.boj" - 2>/dev/null)
if [ "$OUTPUT" = '"-Infinity"' ]; then
    pass "-f stringify: -Infinity becomes string"
else
    fail "-f stringify: -Infinity becomes string (got: $OUTPUT)"
fi

# Test: -f allow with b2b round-trip preserves NaN
./bonbon -f allow b2b "$TMPDIR/nan.boj" "$TMPDIR/nan_rt.boj" 2>/dev/null
if cmp -s "$TMPDIR/nan.boj" "$TMPDIR/nan_rt.boj"; then
    pass "-f allow: b2b preserves NaN"
else
    fail "-f allow: b2b preserves NaN"
fi

# Test: invalid -f mode produces error
if ./bonbon -f invalid b - 2>/dev/null; then
    fail "-f: rejects invalid mode"
else
    pass "-f: rejects invalid mode"
fi

# Summary
echo ""
echo "Results: $PASS passed, $FAIL failed"
if [ "$FAIL" -gt 0 ]; then
    exit 1
fi
