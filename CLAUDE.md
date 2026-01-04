# j2b - JSON/BONJSON Converter

## Purpose

A command-line tool that converts between JSON and BONJSON formats.

## Usage

```
j2b [options] <command> <input> [output]
```

- Use `-` for stdin or stdout
- JSON output is pretty-printed with 4-space indentation
- On BONJSON decode error, outputs whatever was successfully decoded before reporting the error

**Commands:**
- `j` : Validate JSON input (no output)
- `b` : Validate BONJSON input (no output)
- `j2b` : Convert JSON to BONJSON
- `j2j` : Convert JSON to JSON (reformat)
- `b2j` : Convert BONJSON to JSON
- `b2b` : Convert BONJSON to BONJSON (reformat)

**Options:**
- `-e` : Print end offset to stderr (BONJSON input only)
- `-s N` : Skip N bytes before decoding (useful for files with headers)
- `-t` : Allow trailing data (BONJSON input only)

## Architecture

This is a simple single-file CLI application with no complex architecture. All logic is in `main.go`.

### Key Functions

- `main()`: Entry point, handles argument parsing and command dispatch
- `printUsage()`: Prints usage information
- `convert()`: Orchestrates reading, decoding, encoding, and output
- `writeOutput()`: Writes to file or stdout

## Dependencies

- `github.com/kstenerud/go-bonjson`: The BONJSON encoding/decoding library
- Standard library: `encoding/json`, `errors`, `fmt`, `io`, `os`, `strconv`

## Building

```
go build -o j2b
```

## Testing

Run CLI integration tests:
```
./test_cli.sh
```
