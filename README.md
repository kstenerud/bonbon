# bonbon

A command-line tool for working with JSON and BONJSON formats.

[BONJSON](https://github.com/kstenerud/bonjson) is a binary JSON encoding that preserves full JSON fidelity while providing a more compact and energy-efficient representation.

## Installation

```bash
go install github.com/kstenerud/bonbon@latest
```

Or build from source:

```bash
git clone https://github.com/kstenerud/bonbon.git
cd bonbon
go build -o bonbon
```

## Usage

```
bonbon [options] <command> <input> [output]
```

Use `-` for stdin or stdout.

### Commands

| Command | Description                          |
|---------|--------------------------------------|
| `j`     | Validate JSON input (no output)      |
| `b`     | Validate BONJSON input (no output)   |
| `j2b`   | Convert JSON to BONJSON              |
| `j2j`   | Convert JSON to JSON (reformat)      |
| `b2j`   | Convert BONJSON to JSON              |
| `b2b`   | Convert BONJSON to BONJSON (dechunk) |

### Options

| Option | Description                                             |
|--------|---------------------------------------------------------|
| `-e`   | Print end offset to stderr (BONJSON input only)         |
| `-s N` | Skip N bytes before decoding                            |
| `-t`   | Allow trailing data after document (BONJSON input only) |

## Examples

Convert JSON to BONJSON:

```bash
bonbon j2b input.json output.boj
```

Convert BONJSON to JSON:

```bash
bonbon b2j input.boj output.json
```

Validate JSON from stdin:

```bash
echo '{"valid": true}' | bonbon j -
```

Convert and pipe to another tool:

```bash
cat data.json | bonbon j2b - - | hexdump -C
```

Reformat JSON with pretty printing:

```bash
bonbon j2j compact.json pretty.json
```

Skip a header before decoding:

```bash
bonbon -s 16 b2j file-with-header.boj output.json
```

Get the end offset of a BONJSON document:

```bash
bonbon -e b document.boj 2>&1 >/dev/null
```

## Error Handling

When decoding BONJSON, if an error occurs, bonbon outputs whatever was successfully decoded before reporting the error. This allows partial recovery from damaged or corrupted files.

## License

MIT License - see [LICENSE](LICENSE) for details.
