# j2b - JSON/BONJSON Converter

## Purpose

A command-line tool that converts between JSON and BONJSON formats. It automatically detects the input format and converts to the other format.

## Usage

```
j2b <input-file> [output-file]
```

- Use `-` for stdin (input) or stdout (output)
- If `output-file` is omitted, output goes to stdout
- JSON output is pretty-printed with 4-space indentation

## Architecture

This is a simple single-file CLI application with no complex architecture. All logic is in `main.go`.

### Key Functions

- `main()`: Entry point, handles argument parsing and error reporting
- `convert()`: Orchestrates reading, detection, conversion, and output
- `detectJSON()`: Determines if input is JSON or BONJSON by examining byte patterns
- `jsonToBONJSON()`: Converts JSON → BONJSON using the go-bonjson library
- `bonjsonToJSON()`: Converts BONJSON → JSON with pretty printing
- `writeOutput()`: Writes to file or stdout

### Format Detection

Detecting JSON vs BONJSON is non-trivial because many BONJSON type codes overlap with valid JSON start characters. The detection examines subsequent bytes to disambiguate.

**BONJSON type codes (from the spec):**
- `0x00-0x64`: Small integers 0-100
- `0x65-0x67`: Reserved
- `0x68`: Long string
- `0x69`: Big number
- `0x6a-0x6c`: Float types (16/32/64-bit)
- `0x6d`: Null
- `0x6e`: False
- `0x6f`: True
- `0x70-0x77`: Unsigned integers (1-8 bytes)
- `0x78-0x7f`: Signed integers (1-8 bytes)
- `0x80-0x8f`: Short strings (0-15 bytes)
- `0x90-0x98`: Reserved
- `0x99`: Array start
- `0x9a`: Object start
- `0x9b`: Container end
- `0x9c-0xff`: Small integers -100 to -1

**Key overlaps with JSON start characters:**
- `"` (0x22) = BONJSON small int 34
- `-` (0x2d) = BONJSON small int 45
- `0`-`9` (0x30-0x39) = BONJSON small ints 48-57
- `[` (0x5b) = BONJSON small int 91
- `n` (0x6e) = BONJSON false
- `t` (0x74) = BONJSON unsigned 5-byte int type
- `{` (0x7b) = BONJSON signed 4-byte int type

**Unambiguous cases:**
- `f` (0x66) → JSON only (reserved in BONJSON)
- `0x99`, `0x9a` → BONJSON only (not valid ASCII for JSON)
- Any non-JSON-start byte → BONJSON

**Disambiguation by context:**
- `t` followed by `rue` → JSON true
- `n` followed by `ull` → JSON null
- `{` followed by `"` or `}` (after whitespace) → JSON object
- `[` followed by value start or `]` (after whitespace) → JSON array
- `"` followed by more data → JSON string
- `-` followed by digit → JSON negative number
- Digit followed by number continuation → JSON number
- Single digit with no continuation → BONJSON small integer

## Dependencies

- `github.com/kstenerud/go-bonjson`: The BONJSON encoding/decoding library
- Standard library: `encoding/json`, `fmt`, `io`, `os`

## Building

```
go build -o j2b
```

## Testing

Run unit tests:
```
go test -v
```

Manual round-trip test:
```
echo '{"hello": "world"}' > test.json
./j2b test.json test.boj
./j2b test.boj
```
