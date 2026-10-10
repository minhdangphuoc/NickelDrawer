package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

// The shared paging vectors (lib/copperui/tests/pack-vectors.json) are read by the Go tests of
// cmd/ndpreview directly; this file turns the same vectors into the C++ table
// lib/copperui/tests/pack_vectors.inc for the library's host tests.

const maxPackBlocks = 12

type packVector struct {
	Name      string   `json:"name"`
	Available int      `json:"available"`
	Blocks    [][3]int `json:"blocks"`
	Pages     []int    `json:"pages"`
}

type packVectorFile struct {
	Comment string       `json:"comment"`
	Cases   []packVector `json:"cases"`
}

func loadPackVectors(path string) (*packVectorFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read the pack vectors %s: %w", path, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var file packVectorFile
	if err := decoder.Decode(&file); err != nil {
		return nil, fmt.Errorf("the pack vectors %s are not valid: %w", path, err)
	}
	for _, vector := range file.Cases {
		if len(vector.Blocks) > maxPackBlocks {
			return nil, fmt.Errorf("pack vector %q has %d blocks; the C++ table holds %d", vector.Name, len(vector.Blocks), maxPackBlocks)
		}
		if len(vector.Pages) != len(vector.Blocks) {
			return nil, fmt.Errorf("pack vector %q lists %d pages for %d blocks; give one page per block", vector.Name, len(vector.Pages), len(vector.Blocks))
		}
	}
	return &file, nil
}

func emitPackVectors(file *packVectorFile) []byte {
	var out bytes.Buffer
	fmt.Fprintf(&out, "// %s\n", generatedNote)
	out.WriteString("// Source: lib/copperui/tests/pack-vectors.json\n\n")
	fmt.Fprintf(&out, "static const int MAX_PACK_BLOCKS = %d;\n\n", maxPackBlocks)
	out.WriteString("static const PackVector PACK_VECTORS[] = {\n")
	for _, vector := range file.Cases {
		fmt.Fprintf(&out, "    {%s, %d, %d, {", cString(vector.Name), vector.Available, len(vector.Blocks))
		for index, block := range vector.Blocks {
			if index > 0 {
				out.WriteString(", ")
			}
			fmt.Fprintf(&out, "{%d, %d, %d}", block[0], block[1], block[2])
		}
		out.WriteString("}, {")
		for index, page := range vector.Pages {
			if index > 0 {
				out.WriteString(", ")
			}
			fmt.Fprintf(&out, "%d", page)
		}
		out.WriteString("}},\n")
	}
	out.WriteString("};\n")
	return out.Bytes()
}
