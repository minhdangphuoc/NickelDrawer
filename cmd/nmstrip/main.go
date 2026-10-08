// Command nmstrip removes the legacy hand-written LocalSend items from a NickelMenu items file.
package main

import (
	"fmt"
	"os"

	"koboapps/internal/nmconfig"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: nmstrip <items-file>   (rewrites the file in place)")
		os.Exit(2)
	}
	if err := stripFile(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "nmstrip:", err)
		os.Exit(1)
	}
}

// stripFile rewrites via a sibling temp file so an interrupted run cannot truncate the items file.
func stripFile(itemsPath string) error {
	original, err := os.ReadFile(itemsPath)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", itemsPath, err)
	}
	cleaned := nmconfig.RemoveLegacyLocalSendItems(string(original))
	if cleaned == string(original) {
		return nil
	}
	tempPath := itemsPath + ".nmstrip.tmp"
	if err := os.WriteFile(tempPath, []byte(cleaned), 0o644); err != nil {
		return fmt.Errorf("cannot write %s: %w", tempPath, err)
	}
	if err := os.Rename(tempPath, itemsPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("cannot replace %s: %w", itemsPath, err)
	}
	return nil
}
