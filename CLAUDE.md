# bonbon - JSON/BONJSON Converter

## Purpose

A command-line tool for working with JSON and BONJSON formats.

## Usage

```
bonbon [options] <command> <input> [output]
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
- `b2b` : Convert BONJSON to BONJSON (dechunk)

**Options:**
- `-d MODE` : Duplicate key handling (BONJSON input only): reject (default), keepfirst, keeplast
- `-e` : Print end offset to stderr (BONJSON input only)
- `-f MODE` : Special float (NaN, Infinity) handling (BONJSON only): reject (default), allow, stringify
- `-n` : Allow NUL characters in strings (BONJSON input only)
- `-s N` : Skip N bytes before decoding (useful for files with headers)
- `-t` : Allow trailing data (BONJSON input only)
- `-u MODE` : Invalid UTF-8 handling (BONJSON input only): reject (default), replace, delete, ignore

## Architecture

This is a simple single-file CLI application with no complex architecture. All logic is in `main.go`.

### Key Functions

- `main()`: Entry point, handles argument parsing and command dispatch
- `printUsage()`: Prints usage information
- `convert()`: Orchestrates reading, decoding, encoding, and output
- `writeOutput()`: Writes to file or stdout

## Dependencies

- `github.com/kstenerud/go-bonjson`: The BONJSON encoding/decoding library
- Standard library: `bytes`, `encoding/json`, `errors`, `fmt`, `io`, `os`, `strconv`

## Building

```
go build -o bonbon
```

## Testing

Run CLI integration tests:
```
./test_cli.sh
```
