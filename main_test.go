// ABOUTME: Unit tests for the j2b JSON/BONJSON converter.
// ABOUTME: Focuses on format detection since that's the non-trivial logic.

package main

import (
	"testing"
)

func TestDetectJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		wantJSON bool
	}{
		// === Unambiguously JSON ===

		// 'f' (0x66) is reserved in BONJSON, so unambiguously JSON
		{"false literal", []byte("false"), true},
		{"false with whitespace", []byte("  false"), true},

		// === Unambiguously BONJSON ===

		// 0x99 is BONJSON array start (not valid JSON character)
		{"BONJSON array start", []byte{0x99, 0x01, 0x02, 0x9b}, false},

		// 0x9a is BONJSON object start (not valid JSON character)
		{"BONJSON object start", []byte{0x9a, 0x81, 'a', 0x01, 0x9b}, false},

		// 0x00 is BONJSON small int 0 (not valid JSON start)
		{"BONJSON small int 0", []byte{0x00}, false},

		// 0x64 is BONJSON small int 100 (not valid JSON start, 'd' in ASCII)
		{"BONJSON small int 100", []byte{0x64}, false},

		// 0x6d is BONJSON null (not valid JSON start, 'm' in ASCII)
		{"BONJSON null", []byte{0x6d}, false},

		// 0x6e is BONJSON false (could be 'n' but alone it's BONJSON)
		{"BONJSON false standalone", []byte{0x6e}, false},

		// 0x6f is BONJSON true (not valid JSON start, 'o' in ASCII)
		{"BONJSON true", []byte{0x6f}, false},

		// 0x80 is BONJSON short string of length 0
		{"BONJSON empty short string", []byte{0x80}, false},

		// 0x85 followed by 5 bytes is BONJSON short string
		{"BONJSON short string hello", []byte{0x85, 'h', 'e', 'l', 'l', 'o'}, false},

		// 0x9c-0xff are BONJSON negative small integers
		{"BONJSON small int -1", []byte{0xff}, false},
		{"BONJSON small int -100", []byte{0x9c}, false},

		// === Disambiguation: 't' (JSON true vs BONJSON uint 5-byte) ===

		{"JSON true", []byte("true"), true},
		{"JSON true with trailing", []byte("true,"), true},
		// 't' not followed by 'rue' - must be BONJSON unsigned 5-byte int
		{"BONJSON uint5 starting with t", []byte{'t', 0x00, 0x00, 0x00, 0x00, 0x00}, false},
		{"t alone", []byte{'t'}, false},
		{"t followed by wrong char", []byte{'t', 'x'}, false},

		// === Disambiguation: 'n' (JSON null vs BONJSON false) ===

		{"JSON null", []byte("null"), true},
		{"JSON null with trailing", []byte("null}"), true},
		// 'n' not followed by 'ull' - must be BONJSON false
		{"n alone", []byte{'n'}, false},
		{"n followed by wrong char", []byte{'n', 'o'}, false},

		// === Disambiguation: '{' (JSON object vs BONJSON signed 4-byte int) ===

		{"JSON empty object", []byte("{}"), true},
		{"JSON object with key", []byte(`{"key": 1}`), true},
		{"JSON object with whitespace before key", []byte("{ \"key\": 1}"), true},
		{"JSON object with newline before key", []byte("{\n\"key\": 1}"), true},
		{"JSON object with tab before key", []byte("{\t\"key\": 1}"), true},
		{"JSON empty object with whitespace", []byte("{  }"), true},
		// '{' followed by 4 bytes that don't look like JSON object
		{"BONJSON signed int starting with brace", []byte{'{', 0x01, 0x02, 0x03, 0x04}, false},

		// === Disambiguation: '[' (JSON array vs BONJSON small int 91) ===

		{"JSON empty array", []byte("[]"), true},
		{"JSON array with number", []byte("[1]"), true},
		{"JSON array with string", []byte(`["a"]`), true},
		{"JSON array with whitespace", []byte("[ 1 ]"), true},
		{"JSON array with nested array", []byte("[[]]"), true},
		{"JSON array with nested object", []byte("[{}]"), true},
		{"JSON array with true", []byte("[true]"), true},
		{"JSON array with false", []byte("[false]"), true},
		{"JSON array with null", []byte("[null]"), true},
		{"JSON array with negative", []byte("[-1]"), true},
		// '[' alone is BONJSON small int 91
		{"bracket alone", []byte{'['}, false},

		// === Disambiguation: '"' (JSON string vs BONJSON small int 34) ===

		{"JSON string", []byte(`"hello"`), true},
		{"JSON empty string", []byte(`""`), true},
		// '"' alone is BONJSON small int 34
		{"quote alone", []byte{'"'}, false},

		// === Disambiguation: '-' (JSON negative number vs BONJSON small int 45) ===

		{"JSON negative number", []byte("-5"), true},
		{"JSON negative zero", []byte("-0"), true},
		{"JSON negative float", []byte("-3.14"), true},
		// '-' alone is BONJSON small int 45
		{"minus alone", []byte{'-'}, false},
		// '-' followed by non-digit is BONJSON
		{"minus followed by letter", []byte{'-', 'a'}, false},

		// === Disambiguation: digits (JSON number vs BONJSON small int) ===

		// Single digit defaults to BONJSON
		{"single digit 0", []byte("0"), false},
		{"single digit 5", []byte("5"), false},
		{"single digit 9", []byte("9"), false},

		// Digit followed by more digits is JSON
		{"multi-digit number", []byte("123"), true},
		{"number with decimal", []byte("1.5"), true},
		{"number with exponent", []byte("1e10"), true},
		{"number with capital E", []byte("1E10"), true},
		{"number with trailing whitespace", []byte("1 "), true},
		{"number with trailing comma", []byte("1,"), true},
		{"number with trailing bracket", []byte("1]"), true},
		{"number with trailing brace", []byte("1}"), true},

		// === Edge cases ===

		{"only whitespace", []byte("   "), true}, // defaults to JSON (will error on parse)
		{"whitespace then JSON", []byte("  true"), true},
		{"whitespace then BONJSON", []byte{' ', 0x99, 0x9b}, false},

		// Leading whitespace before BONJSON - whitespace chars are valid BONJSON ints
		// but we skip whitespace looking for content, then detect based on next byte
		{"tab then BONJSON array", []byte{'\t', 0x99, 0x9b}, false},
		{"newline then BONJSON object", []byte{'\n', 0x9a, 0x9b}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectJSON(tt.input)
			if got != tt.wantJSON {
				if tt.wantJSON {
					t.Errorf("detectJSON(%v) = false (BONJSON), want true (JSON)", tt.input)
				} else {
					t.Errorf("detectJSON(%v) = true (JSON), want false (BONJSON)", tt.input)
				}
			}
		})
	}
}

func TestIsWhitespace(t *testing.T) {
	whitespace := []byte{' ', '\t', '\n', '\r'}
	for _, b := range whitespace {
		if !isWhitespace(b) {
			t.Errorf("isWhitespace(%q) = false, want true", b)
		}
	}

	nonWhitespace := []byte{'a', '0', '{', '[', 0x00, 0xff}
	for _, b := range nonWhitespace {
		if isWhitespace(b) {
			t.Errorf("isWhitespace(%q) = true, want false", b)
		}
	}
}

func TestIsDigit(t *testing.T) {
	for b := byte('0'); b <= '9'; b++ {
		if !isDigit(b) {
			t.Errorf("isDigit(%q) = false, want true", b)
		}
	}

	nonDigits := []byte{'a', 'z', '{', ' ', 0x00}
	for _, b := range nonDigits {
		if isDigit(b) {
			t.Errorf("isDigit(%q) = true, want false", b)
		}
	}
}

func TestIsValidJSONStart(t *testing.T) {
	valid := []byte{'{', '[', '"', 't', 'f', 'n', '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}
	for _, b := range valid {
		if !isValidJSONStart(b) {
			t.Errorf("isValidJSONStart(%q) = false, want true", b)
		}
	}

	invalid := []byte{'a', 'z', ' ', '\t', 0x00, 0x99, 0x9a}
	for _, b := range invalid {
		if isValidJSONStart(b) {
			t.Errorf("isValidJSONStart(%q) = true, want false", b)
		}
	}
}

func TestLooksLikeJSONObject(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		wantJSON bool
	}{
		{"empty object", []byte("}"), true},
		{"key start", []byte(`"key"`), true},
		{"whitespace then key", []byte(`  "key"`), true},
		{"whitespace then close", []byte("  }"), true},
		{"binary data", []byte{0x01, 0x02, 0x03, 0x04}, false},
		{"empty", []byte{}, false},
		{"wrong char", []byte("abc"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := looksLikeJSONObject(tt.input)
			if got != tt.wantJSON {
				t.Errorf("looksLikeJSONObject(%v) = %v, want %v", tt.input, got, tt.wantJSON)
			}
		})
	}
}

func TestLooksLikeJSONArray(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		wantJSON bool
	}{
		{"empty array", []byte("]"), true},
		{"number element", []byte("1]"), true},
		{"string element", []byte(`"a"]`), true},
		{"object element", []byte("{}]"), true},
		{"array element", []byte("[]"), true},
		{"true element", []byte("true]"), true},
		{"false element", []byte("false]"), true},
		{"null element", []byte("null]"), true},
		{"negative element", []byte("-1]"), true},
		{"whitespace then element", []byte("  1]"), true},
		{"whitespace then close", []byte("  ]"), true},
		{"empty", []byte{}, false},
		{"wrong char", []byte("abc"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := looksLikeJSONArray(tt.input)
			if got != tt.wantJSON {
				t.Errorf("looksLikeJSONArray(%v) = %v, want %v", tt.input, got, tt.wantJSON)
			}
		})
	}
}

func TestIsJSONNumberOrDocContinuation(t *testing.T) {
	valid := []byte{'0', '1', '9', '.', 'e', 'E', ' ', '\t', '\n', '\r', ',', ']', '}'}
	for _, b := range valid {
		if !isJSONNumberOrDocContinuation(b) {
			t.Errorf("isJSONNumberOrDocContinuation(%q) = false, want true", b)
		}
	}

	invalid := []byte{'a', 'x', '{', '[', '"', 0x00}
	for _, b := range invalid {
		if isJSONNumberOrDocContinuation(b) {
			t.Errorf("isJSONNumberOrDocContinuation(%q) = true, want false", b)
		}
	}
}
