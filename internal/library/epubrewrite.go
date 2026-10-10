package library

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

// epubRewrite describes one kind of change to an EPUB: how the package document and the
// content documents are rewritten, and what the finished copy must prove before it replaces
// the original.
type epubRewrite struct {
	packageDocument func(content []byte) ([]byte, error)
	contentDocument func(content []byte) []byte // nil: content documents are copied untouched
	verify          func(pkg packageXML) error
}

// rewriteEpubInPlace replaces the file with a rewritten copy, after checking the copy.
func rewriteEpubInPlace(absolute string, rewrite epubRewrite) error {
	return writeAtomic(absolute, func(temp *os.File) error {
		if err := writeRewrittenEpub(absolute, temp, rewrite); err != nil {
			return err
		}
		if _, err := temp.Seek(0, io.SeekStart); err != nil {
			return err
		}
		return verifyRewrittenEpub(temp, rewrite)
	})
}

// writeRewrittenEpub copies every entry untouched (raw, still compressed) except the package
// document and content documents, which the rewrite changes. mimetype goes first, stored.
func writeRewrittenEpub(source string, output io.Writer, rewrite epubRewrite) error {
	archive, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer archive.Close()
	info, err := ReadEpub(source)
	if err != nil {
		return err
	}
	writer := zip.NewWriter(output)
	for _, file := range orderedEntries(archive.File) {
		if err := copyEntry(writer, file, file.Name == info.OPFPath, rewrite); err != nil {
			return fmt.Errorf("%s: %w", file.Name, err)
		}
	}
	return writer.Close()
}

// orderedEntries puts the stored mimetype entry first, as the EPUB container format requires.
func orderedEntries(files []*zip.File) []*zip.File {
	ordered := make([]*zip.File, 0, len(files))
	for _, file := range files {
		if file.Name == "mimetype" {
			ordered = append(ordered, file)
		}
	}
	for _, file := range files {
		if file.Name != "mimetype" {
			ordered = append(ordered, file)
		}
	}
	return ordered
}

func copyEntry(writer *zip.Writer, file *zip.File, isPackage bool, rewrite epubRewrite) error {
	extension := strings.ToLower(path.Ext(file.Name))
	isRewritten := isPackage || (rewrite.contentDocument != nil && rewritableExtensions[extension])
	if !isRewritten || file.UncompressedSize64 > maxRewriteBytes {
		return writer.Copy(file)
	}
	content, err := readFile(file)
	if err != nil {
		return err
	}
	changed := content
	if isPackage {
		changed, err = rewrite.packageDocument(content)
		if err != nil {
			return err
		}
	} else {
		changed = rewrite.contentDocument(content)
	}
	if bytes.Equal(changed, content) {
		return writer.Copy(file)
	}
	header := zip.FileHeader{Name: file.Name, Method: file.Method, Modified: file.Modified}
	if file.Name == "mimetype" {
		header.Method = zip.Store
	}
	entry, err := writer.CreateHeader(&header)
	if err != nil {
		return err
	}
	_, err = entry.Write(changed)
	return err
}

func readFile(file *zip.File) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(io.LimitReader(reader, maxRewriteBytes+1))
}

// verifyRewrittenEpub re-reads the finished copy: every entry must decompress, mimetype must
// be first and stored, and the package document must satisfy the rewrite's own check.
func verifyRewrittenEpub(file *os.File, rewrite epubRewrite) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	archive, err := zip.NewReader(file, info.Size())
	if err != nil {
		return fmt.Errorf("rewritten file is not a zip: %w", err)
	}
	if len(archive.File) == 0 || archive.File[0].Name != "mimetype" || archive.File[0].Method != zip.Store {
		return errors.New("rewritten file lacks a stored mimetype entry first")
	}
	for _, entry := range archive.File {
		if err := drain(entry); err != nil {
			return fmt.Errorf("rewritten entry %s: %w", entry.Name, err)
		}
	}
	return verifyPackage(archive, rewrite.verify)
}

func drain(entry *zip.File) error {
	reader, err := entry.Open()
	if err != nil {
		return err
	}
	defer reader.Close()
	_, err = io.Copy(io.Discard, reader)
	return err
}

func verifyPackage(archive *zip.Reader, check func(pkg packageXML) error) error {
	entries := indexEntries(archive.File)
	opfPath, err := findOPFPath(entries)
	if err != nil {
		return err
	}
	var pkg packageXML
	if err := decodeEntry(entries, opfPath, maxOPFSize, &pkg); err != nil {
		return err
	}
	return check(pkg)
}
