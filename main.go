// ABOUTME: CLI tool that converts between JSON and BONJSON formats.
// ABOUTME: Automatically detects input format and converts to the other format.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/kstenerud/go-bonjson"
)

func main() {
	var allowTrailing bool
	var skipBytes int
	args := os.Args[1:]

	// Parse flags
	for len(args) > 0 && len(args[0]) > 0 && args[0][0] == '-' && args[0] != "-" {
		switch args[0] {
		case "-t":
			allowTrailing = true
			args = args[1:]
		case "-s", "--start":
			if len(args) < 2 {
				fmt.Fprintln(os.Stderr, "Error: -s requires an argument")
				os.Exit(1)
			}
			var err error
			skipBytes, err = strconv.Atoi(args[1])
			if err != nil || skipBytes < 0 {
				fmt.Fprintf(os.Stderr, "Error: invalid skip value: %s\n", args[1])
				os.Exit(1)
			}
			args = args[2:]
		default:
			fmt.Fprintf(os.Stderr, "Unknown option: %s\n", args[0])
			os.Exit(1)
		}
	}

	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: j2b [-t] [-s N] <input-file> <output-file>")
		fmt.Fprintln(os.Stderr, "  Converts JSON to BONJSON or BONJSON to JSON.")
		fmt.Fprintln(os.Stderr, "  Format is auto-detected from the input.")
		fmt.Fprintln(os.Stderr, "  Use '-' for stdin/stdout.")
		fmt.Fprintln(os.Stderr, "Options:")
		fmt.Fprintln(os.Stderr, "  -t       Allow trailing data after BONJSON document")
		fmt.Fprintln(os.Stderr, "  -s N     Skip N bytes before decoding")
		os.Exit(1)
	}

	inputPath := args[0]
	outputPath := args[1]

	if err := convert(inputPath, outputPath, allowTrailing, skipBytes); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// convert reads the input, detects its format, and converts it to the other
// format. If inputPath is "-", reads from stdin. If outputPath is empty or
// "-", output goes to stdout. If allowTrailing is true, trailing data after
// a BONJSON document is ignored. If skipBytes > 0, that many bytes are skipped
// before decoding.
func convert(inputPath, outputPath string, allowTrailing bool, skipBytes int) error {
	var data []byte
	var err error
	if inputPath == "-" {
		data, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
	} else {
		data, err = os.ReadFile(inputPath)
		if err != nil {
			return fmt.Errorf("reading input file: %w", err)
		}
	}

	if skipBytes > 0 {
		if skipBytes >= len(data) {
			return fmt.Errorf("skip value %d exceeds input size %d", skipBytes, len(data))
		}
		data = data[skipBytes:]
	}

	if len(data) == 0 {
		return fmt.Errorf("input is empty")
	}

	isJSON := detectJSON(data)

	var output []byte
	var outputIsJSON bool
	if isJSON {
		output, err = jsonToBONJSON(data)
		if err != nil {
			return fmt.Errorf("converting JSON to BONJSON: %w", err)
		}
		outputIsJSON = false
	} else {
		output, err = bonjsonToJSON(data, allowTrailing)
		if err != nil {
			return fmt.Errorf("converting BONJSON to JSON: %w", err)
		}
		outputIsJSON = true
	}

	return writeOutput(output, outputPath, outputIsJSON)
}

// detectJSON determines if the data appears to be JSON (text) or BONJSON (binary).
// This is non-trivial because many BONJSON type codes overlap with valid JSON start
// characters. The detection looks at subsequent bytes to disambiguate.
//
// Key observations from the BONJSON spec:
//   - 0x99 (array start) and 0x9a (object start) are unambiguously BONJSON
//   - 0x66 ('f') is reserved in BONJSON, so unambiguously JSON
//   - Small integers 0-100 (0x00-0x64) overlap with many ASCII chars including digits
//   - 0x6e is BONJSON false, but also ASCII 'n' (start of JSON null)
//   - 0x74 is BONJSON unsigned 5-byte int type, but also ASCII 't' (start of JSON true)
//   - 0x7b is BONJSON signed 4-byte int type, but also ASCII '{' (JSON object start)
func detectJSON(data []byte) bool {
	// Skip leading whitespace
	start := skipWhitespace(data, 0)
	if start >= len(data) {
		return true // Only whitespace, default to JSON (will error on parse)
	}

	first := data[start]

	// Unambiguously BONJSON: container starts that aren't valid ASCII for JSON
	if first == 0x99 || first == 0x9a {
		return false
	}

	// Unambiguously JSON: 'f' (0x66 is reserved in BONJSON)
	if first == 'f' {
		return true
	}

	// If it's not a valid JSON start character, it must be BONJSON
	if !isValidJSONStart(first) {
		return false
	}

	// For ambiguous bytes, examine subsequent bytes to disambiguate
	remaining := data[start+1:]

	switch first {
	case 't':
		// JSON true: must be followed by "rue"
		// BONJSON: unsigned 5-byte integer (type code followed by 5 bytes of data)
		return len(remaining) >= 3 &&
			remaining[0] == 'r' && remaining[1] == 'u' && remaining[2] == 'e'

	case 'n':
		// JSON null: must be followed by "ull"
		// BONJSON: false (single byte, document complete)
		return len(remaining) >= 3 &&
			remaining[0] == 'u' && remaining[1] == 'l' && remaining[2] == 'l'

	case '{':
		// JSON object: { followed by optional whitespace, then " or }
		// BONJSON: signed 4-byte integer (type code followed by 4 bytes of data)
		return looksLikeJSONObject(remaining)

	case '[':
		// JSON array: [ followed by optional whitespace, then value or ]
		// BONJSON: small integer 91 (single byte, document complete)
		return looksLikeJSONArray(remaining)

	case '"':
		// JSON string: " followed by string content and closing "
		// BONJSON: small integer 34 (single byte, document complete)
		// If there's more data, it's almost certainly JSON
		return len(remaining) > 0

	case '-':
		// JSON negative number: - must be followed by a digit
		// BONJSON: small integer 45 (single byte, document complete)
		return len(remaining) > 0 && isDigit(remaining[0])

	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		// JSON number: digit optionally followed by more digits, decimal, exponent
		// BONJSON: small integer 48-57 (single byte, document complete)
		if len(remaining) == 0 {
			// Single digit is ambiguous; default to BONJSON since it's more
			// likely someone is converting a BONJSON small int than a JSON
			// document containing just a single digit
			return false
		}
		// If followed by valid JSON number/document continuation, it's JSON
		return isJSONNumberOrDocContinuation(remaining[0])
	}

	// Default to JSON for any unhandled case
	return true
}

// skipWhitespace returns the index of the first non-whitespace byte at or after start.
func skipWhitespace(data []byte, start int) int {
	for start < len(data) && isWhitespace(data[start]) {
		start++
	}
	return start
}

// isWhitespace returns true if b is a JSON whitespace character.
func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// isDigit returns true if b is an ASCII digit.
func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

// isValidJSONStart returns true if b can be the first non-whitespace byte of a JSON document.
func isValidJSONStart(b byte) bool {
	switch b {
	case '{', '[', '"', 't', 'f', 'n', '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return true
	}
	return false
}

// looksLikeJSONObject checks if remaining bytes (after '{') look like a JSON object.
// In JSON, after '{' we expect optional whitespace then '"' (key) or '}' (empty object).
func looksLikeJSONObject(remaining []byte) bool {
	i := skipWhitespace(remaining, 0)
	if i >= len(remaining) {
		return false // EOF after '{', not valid JSON but also not 4-byte BONJSON int
	}
	return remaining[i] == '"' || remaining[i] == '}'
}

// looksLikeJSONArray checks if remaining bytes (after '[') look like a JSON array.
// In JSON, after '[' we expect optional whitespace then a value start or ']' (empty array).
func looksLikeJSONArray(remaining []byte) bool {
	i := skipWhitespace(remaining, 0)
	if i >= len(remaining) {
		return false // EOF after '[', not valid but lean toward BONJSON (int 91)
	}
	// Check for valid JSON array content: value start or closing bracket
	return isValidJSONStart(remaining[i]) || remaining[i] == ']'
}

// isJSONNumberOrDocContinuation returns true if b could follow a digit in JSON.
// This includes more digits, decimal point, exponent, or structural characters.
func isJSONNumberOrDocContinuation(b byte) bool {
	switch b {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9': // more digits
		return true
	case '.': // decimal point
		return true
	case 'e', 'E': // exponent
		return true
	case ' ', '\t', '\n', '\r': // whitespace after number
		return true
	case ',', ']', '}': // structural characters after number
		return true
	}
	return false
}

// jsonToBONJSON converts JSON data to BONJSON format.
func jsonToBONJSON(data []byte) ([]byte, error) {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	return bonjson.Marshal(value)
}

// bonjsonToJSON converts BONJSON data to pretty-printed JSON format.
// If allowTrailing is true, trailing data after the BONJSON document is ignored.
func bonjsonToJSON(data []byte, allowTrailing bool) ([]byte, error) {
	var value any
	_, err := bonjson.UnmarshalWithByteCount(data, &value)
	if err != nil {
		// If trailing data error and we're allowing it, ignore the error
		// since the value was successfully decoded
		var trailingErr *bonjson.TrailingDataError
		if allowTrailing && errors.As(err, &trailingErr) {
			err = nil
		}
	}
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(value, "", "    ")
}

// writeOutput writes data to the specified file, or to stdout if path is empty
// or "-". When outputting JSON to stdout, a trailing newline is added for
// better terminal display.
func writeOutput(data []byte, outputPath string, isJSON bool) error {
	var w io.Writer
	if outputPath == "" || outputPath == "-" {
		w = os.Stdout
	} else {
		f, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("creating output file: %w", err)
		}
		defer f.Close()
		w = f
	}

	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	// Add trailing newline for JSON output to stdout for better terminal display
	if outputPath == "" && isJSON {
		fmt.Fprintln(w)
	}

	return nil
}
