package library

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
)

const (
	maxEntries       = 20000
	maxContainerSize = 256 << 10
	maxOPFSize       = 4 << 20
	// sampleBytes is how much body text feeds the language guess.
	sampleBytes = 20 << 10
	// perDocumentBytes spreads the sample over several chapters instead of one long cover page.
	perDocumentBytes = 6 << 10
	// documentReadBytes is how much raw markup is read per chapter to get perDocumentBytes of text.
	documentReadBytes = 96 << 10

	containerEntry = "META-INF/container.xml"
)

// EpubInfo is what the scan learns from one EPUB without unpacking it.
type EpubInfo struct {
	OPFPath  string
	Title    string
	Author   string
	Language string
	Sample   string // visible text from the first chapters, about sampleBytes
}

type containerXML struct {
	Rootfiles []struct {
		FullPath string `xml:"full-path,attr"`
	} `xml:"rootfiles>rootfile"`
}

type packageXML struct {
	Titles    []string `xml:"metadata>title"`
	Creators  []string `xml:"metadata>creator"`
	Languages []string `xml:"metadata>language"`
	Items     []struct {
		ID        string `xml:"id,attr"`
		Href      string `xml:"href,attr"`
		MediaType string `xml:"media-type,attr"`
	} `xml:"manifest>item"`
	Spine []struct {
		IDRef string `xml:"idref,attr"`
	} `xml:"spine>itemref"`
}

// ReadEpub streams the OPF and the first chapters out of the zip; the book is never unpacked.
func ReadEpub(filePath string) (EpubInfo, error) {
	archive, err := zip.OpenReader(filePath)
	if err != nil {
		return EpubInfo{}, fmt.Errorf("not a readable zip: %w", err)
	}
	defer archive.Close()
	if len(archive.File) > maxEntries {
		return EpubInfo{}, fmt.Errorf("more than %d entries", maxEntries)
	}
	entries := indexEntries(archive.File)
	opfPath, err := findOPFPath(entries)
	if err != nil {
		return EpubInfo{}, err
	}
	var pkg packageXML
	if err := decodeEntry(entries, opfPath, maxOPFSize, &pkg); err != nil {
		return EpubInfo{}, fmt.Errorf("package document %s: %w", opfPath, err)
	}
	info := EpubInfo{
		OPFPath: opfPath, Title: firstText(pkg.Titles), Author: firstText(pkg.Creators),
		Language: firstText(pkg.Languages),
	}
	info.Sample = sampleText(entries, opfPath, pkg)
	return info, nil
}

func indexEntries(files []*zip.File) map[string]*zip.File {
	entries := make(map[string]*zip.File, len(files))
	for _, file := range files {
		entries[file.Name] = file
	}
	return entries
}

func findOPFPath(entries map[string]*zip.File) (string, error) {
	var container containerXML
	if err := decodeEntry(entries, containerEntry, maxContainerSize, &container); err != nil {
		return "", fmt.Errorf("%s: %w", containerEntry, err)
	}
	for _, rootfile := range container.Rootfiles {
		opfPath := strings.TrimPrefix(rootfile.FullPath, "/")
		if opfPath != "" && path.Clean(opfPath) == opfPath && !strings.HasPrefix(opfPath, "../") {
			return opfPath, nil
		}
	}
	return "", errors.New("container.xml has no usable rootfile")
}

func decodeEntry(entries map[string]*zip.File, name string, limit int64, target any) error {
	content, err := readEntry(entries, name, limit)
	if err != nil {
		return err
	}
	decoder := xml.NewDecoder(bytes.NewReader(content))
	decoder.Strict = false
	decoder.Entity = xml.HTMLEntity
	return decoder.Decode(target)
}

// readEntry reads at most limit bytes of an entry, which also bounds a zip bomb.
func readEntry(entries map[string]*zip.File, name string, limit int64) ([]byte, error) {
	file, found := entries[name]
	if !found {
		return nil, errors.New("missing from the archive")
	}
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	content, err := io.ReadAll(io.LimitReader(reader, limit))
	if err != nil && len(content) == 0 {
		return nil, err
	}
	return content, nil
}

func firstText(values []string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// sampleText collects visible text from the spine documents in reading order.
func sampleText(entries map[string]*zip.File, opfPath string, pkg packageXML) string {
	hrefs := make(map[string]string)
	for _, item := range pkg.Items {
		if isMarkupType(item.MediaType) {
			hrefs[item.ID] = item.Href
		}
	}
	var sample strings.Builder
	for _, reference := range pkg.Spine {
		if sample.Len() >= sampleBytes {
			break
		}
		name, ok := documentName(opfPath, hrefs[reference.IDRef])
		if !ok {
			continue
		}
		markup, err := readEntry(entries, name, documentReadBytes)
		if err != nil {
			continue
		}
		text := StripTags(markup)
		sample.WriteString(cutRunes(strings.Join(strings.Fields(text), " "), perDocumentBytes))
		sample.WriteByte(' ')
	}
	return sample.String()
}

func isMarkupType(mediaType string) bool {
	return mediaType == "application/xhtml+xml" || mediaType == "text/html"
}

func documentName(opfPath, href string) (string, bool) {
	if href == "" {
		return "", false
	}
	unescaped, err := url.PathUnescape(href)
	if err != nil {
		return "", false
	}
	unescaped, _, _ = strings.Cut(unescaped, "#")
	name := path.Join(path.Dir(opfPath), unescaped)
	return name, !strings.HasPrefix(name, "../")
}

// cutRunes shortens text to about limit bytes at a character boundary.
func cutRunes(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	return strings.ToValidUTF8(text[:limit], "")
}
