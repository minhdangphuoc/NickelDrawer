package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"koboapps/internal/release"
)

// runPackApps writes one zip per app folder under <tree>/.adds/apps. The zips hold no signing
// and no secrets; `sign` indexes them afterwards.
func runPackApps(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("pack-apps", flag.ContinueOnError)
	flags.SetOutput(stderr)
	tree := flags.String("tree", "build/apps-tree", "tree that mirrors the Kobo drive root and holds .adds/apps/<id>")
	outDir := flags.String("out", "dist/apps", "output folder for the app zips")
	if err := flags.Parse(args); err != nil {
		return exitUsage
	}
	zips, err := packApps(*tree, *outDir)
	if err != nil {
		return fail(stderr, err)
	}
	for _, zipPath := range zips {
		fmt.Fprintln(stdout, zipPath)
	}
	return exitOK
}

func packApps(tree, outDir string) ([]string, error) {
	appIDs, err := listAppIDs(tree)
	if err != nil {
		return nil, err
	}
	if err := os.RemoveAll(outDir); err != nil {
		return nil, fmt.Errorf("cannot clear %s: %w", outDir, err)
	}
	var zips []string
	for _, id := range appIDs {
		zipPath, err := release.PackApp(tree, id, outDir)
		if err != nil {
			return nil, err
		}
		zips = append(zips, zipPath)
	}
	return zips, nil
}

func listAppIDs(tree string) ([]string, error) {
	appsDir := filepath.Join(tree, ".adds", "apps")
	entries, err := os.ReadDir(appsDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w; run `make dist` to stage the apps", appsDir, err)
	}
	var ids []string
	for _, entry := range entries {
		if entry.IsDir() {
			ids = append(ids, entry.Name())
		}
	}
	sort.Strings(ids)
	return ids, nil
}
