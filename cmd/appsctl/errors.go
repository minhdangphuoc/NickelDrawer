package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"koboapps/internal/i18n"
)

const (
	maxErrorLines    = 12 // what fits in a NickelMenu dialog
	maxErrorLineRune = 60
)

func runErrors(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("errors", flag.ContinueOnError)
	flags.SetOutput(stderr)
	errorsFile := flags.String("errors-file", defaultErrorsFile, "file written by gen-menu")
	var localeOptions localeFlags
	localeOptions.register(flags)
	if err := flags.Parse(args); err != nil {
		return exitFailure
	}
	bundle, bundleErr := localeOptions.hostBundle()
	if bundleErr != nil {
		fmt.Fprintln(stderr, "warning:", bundleErr)
	}
	lines, err := readErrorLines(*errorsFile)
	if err != nil {
		fmt.Fprintln(stdout, bundle.Message("errors.unreadable", nil))
		fmt.Fprintln(stderr, err)
		return exitFailure
	}
	fmt.Fprint(stdout, formatErrorDialog(lines, bundle))
	return exitOK
}

func readErrorLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil // nothing generated yet means nothing failed
	}
	if err != nil {
		return nil, fmt.Errorf("cannot open %s: %w", path, err)
	}
	defer file.Close()
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" {
			lines = append(lines, line)
		}
	}
	return lines, scanner.Err()
}

func formatErrorDialog(lines []string, bundle *i18n.Bundle) string {
	if len(lines) == 0 {
		return bundle.Message("errors.none", nil) + "\n"
	}
	var text strings.Builder
	for i, line := range lines {
		if i == maxErrorLines-1 && len(lines) > maxErrorLines {
			text.WriteString(bundle.Message("errors.more", i18n.Params("count", strconv.Itoa(len(lines)-i))) + "\n")
			break
		}
		text.WriteString(truncateRunes(line, maxErrorLineRune) + "\n")
	}
	return text.String()
}

func truncateRunes(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit-1]) + "…"
}
