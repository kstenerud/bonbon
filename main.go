// ABOUTME: bonbon - CLI tool for working with JSON and BONJSON formats.
// ABOUTME: Uses explicit commands to specify input/output formats.

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

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: bonbon [options] <command> <input> [output]")
	fmt.Fprintln(os.Stderr, "  Use '-' for stdin/stdout.")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  j        Validate JSON input (no output)")
	fmt.Fprintln(os.Stderr, "  b        Validate BONJSON input (no output)")
	fmt.Fprintln(os.Stderr, "  j2b      Convert JSON to BONJSON")
	fmt.Fprintln(os.Stderr, "  j2j      Convert JSON to JSON (reformat)")
	fmt.Fprintln(os.Stderr, "  b2j      Convert BONJSON to JSON")
	fmt.Fprintln(os.Stderr, "  b2b      Convert BONJSON to BONJSON (dechunk)")
	fmt.Fprintln(os.Stderr, "Options:")
	fmt.Fprintln(os.Stderr, "  -e       Print end offset to stderr (BONJSON input only)")
	fmt.Fprintln(os.Stderr, "  -s N     Skip N bytes before decoding")
	fmt.Fprintln(os.Stderr, "  -t       Allow trailing data (BONJSON input only)")
}

func main() {
	var allowTrailing bool
	var skipBytes int
	var printEndOffset bool
	args := os.Args[1:]

	// Parse flags
	for len(args) > 0 && len(args[0]) > 0 && args[0][0] == '-' && args[0] != "-" {
		switch args[0] {
		case "-t":
			allowTrailing = true
			args = args[1:]
		case "-e":
			printEndOffset = true
			args = args[1:]
		case "-s":
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
		printUsage()
		os.Exit(1)
	}

	command := args[0]
	inputPath := args[1]
	outputPath := ""

	// Determine input/output formats and required args based on command
	var inputJSON, outputJSON bool
	var needsOutput bool

	switch command {
	case "j":
		inputJSON = true
		needsOutput = false
	case "b":
		inputJSON = false
		needsOutput = false
	case "j2b":
		inputJSON = true
		outputJSON = false
		needsOutput = true
	case "j2j":
		inputJSON = true
		outputJSON = true
		needsOutput = true
	case "b2j":
		inputJSON = false
		outputJSON = true
		needsOutput = true
	case "b2b":
		inputJSON = false
		outputJSON = false
		needsOutput = true
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}

	if needsOutput {
		if len(args) < 3 {
			fmt.Fprintf(os.Stderr, "Error: %s command requires an output file\n", command)
			os.Exit(1)
		}
		outputPath = args[2]
	} else {
		if len(args) > 2 {
			fmt.Fprintf(os.Stderr, "Error: %s command does not accept an output file\n", command)
			os.Exit(1)
		}
	}

	if err := convert(inputPath, outputPath, inputJSON, outputJSON, allowTrailing, skipBytes, printEndOffset); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// convert reads the input and converts it to the specified output format.
// If inputPath is "-", reads from stdin. If outputPath is "-", output goes to
// stdout. If outputPath is empty, only validates the input without producing
// output. inputJSON and outputJSON specify the formats. If allowTrailing is
// true, trailing data after a BONJSON document is ignored. If skipBytes > 0,
// that many bytes are skipped before decoding. If printEndOffset is true and
// input is BONJSON, prints the end offset to stderr.
func convert(inputPath, outputPath string, inputJSON, outputJSON bool, allowTrailing bool, skipBytes int, printEndOffset bool) error {
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

	// Decode input
	var value any
	var byteCount int
	var decodeErr error

	if inputJSON {
		if err := json.Unmarshal(data, &value); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
	} else {
		byteCount, decodeErr = bonjson.UnmarshalWithByteCount(data, &value)
		if decodeErr != nil {
			var trailingErr *bonjson.TrailingDataError
			if allowTrailing && errors.As(decodeErr, &trailingErr) {
				decodeErr = nil
			}
		}
		if printEndOffset {
			fmt.Fprintf(os.Stderr, "%d\n", skipBytes+byteCount)
		}
	}

	// Validate-only mode: no output
	if outputPath == "" {
		if decodeErr != nil {
			return fmt.Errorf("invalid BONJSON: %w", decodeErr)
		}
		return nil
	}

	// Encode output
	var output []byte
	if outputJSON {
		output, err = json.MarshalIndent(value, "", "    ")
		if err != nil {
			return fmt.Errorf("encoding JSON: %w", err)
		}
	} else {
		output, err = bonjson.Marshal(value)
		if err != nil {
			return fmt.Errorf("encoding BONJSON: %w", err)
		}
	}

	// Write output (may be partial on BONJSON decode error)
	if len(output) > 0 {
		if err := writeOutput(output, outputPath, outputJSON); err != nil {
			return err
		}
	}

	// Report any decode error after writing partial output
	if decodeErr != nil {
		return fmt.Errorf("decoding BONJSON: %w", decodeErr)
	}

	return nil
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
