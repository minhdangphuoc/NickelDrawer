// Command copperui-tokens turns the CopperUI token file into the C++ header, the Go constants
// and the documentation table, and the shared text vectors into the mod's test table.
//
//	go run ./cmd/copperui-tokens          # rewrite the generated files (make tokens)
//	go run ./cmd/copperui-tokens -check   # fail when a generated file is stale (make check)
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

const (
	tokensPath  = "lib/copperui/tokens/copperui.tokens.json"
	vectorsPath = "lib/copperui/tests/text-vectors.json"
	packPath    = "lib/copperui/tests/pack-vectors.json"
	packIncPath = "lib/copperui/tests/pack_vectors.inc"
	headerPath  = "lib/copperui/include/copperui/tokens.h"
	goPath      = "internal/copperui/tokens.go"
	docPath     = "docs/copperui-tokens.md"
	incPath     = "mod/drawer/tests/text_vectors.inc"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("copperui-tokens", flag.ContinueOnError)
	flags.SetOutput(stderr)
	isCheck := flags.Bool("check", false, "fail when a generated file differs from what the sources produce")
	root := flags.String("root", ".", "repository root")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	generated, err := generateAll(*root)
	if err != nil {
		fmt.Fprintln(stderr, "copperui-tokens:", err)
		return 1
	}
	if *isCheck {
		return reportStale(*root, generated, stderr)
	}
	return writeAll(*root, generated, stdout, stderr)
}

// generateAll returns every generated file by its path below the repository root.
func generateAll(root string) (map[string][]byte, error) {
	spec, err := loadSpec(filepath.Join(root, tokensPath))
	if err != nil {
		return nil, err
	}
	goSource, err := emitGo(spec)
	if err != nil {
		return nil, err
	}
	vectors, err := loadVectors(filepath.Join(root, vectorsPath))
	if err != nil {
		return nil, err
	}
	packVectors, err := loadPackVectors(filepath.Join(root, packPath))
	if err != nil {
		return nil, err
	}
	return map[string][]byte{
		packIncPath: emitPackVectors(packVectors),
		headerPath:  emitCpp(spec),
		goPath:      goSource,
		docPath:     emitMarkdown(spec),
		incPath:     emitVectors(vectors),
	}, nil
}

func staleFiles(root string, generated map[string][]byte) []string {
	var stale []string
	for _, path := range sortedPaths(generated) {
		current, err := os.ReadFile(filepath.Join(root, path))
		if err != nil || !bytes.Equal(current, generated[path]) {
			stale = append(stale, path)
		}
	}
	return stale
}

func reportStale(root string, generated map[string][]byte, stderr io.Writer) int {
	stale := staleFiles(root, generated)
	for _, path := range stale {
		fmt.Fprintf(stderr, "copperui-tokens: %s is stale; run `make tokens`\n", path)
	}
	if len(stale) > 0 {
		return 1
	}
	return 0
}

func writeAll(root string, generated map[string][]byte, stdout, stderr io.Writer) int {
	for _, path := range sortedPaths(generated) {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			fmt.Fprintf(stderr, "copperui-tokens: cannot create the folder of %s: %v\n", path, err)
			return 1
		}
		if err := os.WriteFile(full, generated[path], 0o644); err != nil {
			fmt.Fprintf(stderr, "copperui-tokens: cannot write %s: %v\n", path, err)
			return 1
		}
		fmt.Fprintln(stdout, "wrote", path)
	}
	return 0
}

func sortedPaths(generated map[string][]byte) []string {
	paths := make([]string, 0, len(generated))
	for path := range generated {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}
