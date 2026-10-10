package ghsync

import (
	"path/filepath"
	"strings"

	"golang.org/x/text/unicode/norm"

	"koboapps/internal/i18n"
)

const maxSegmentBytes = 255

// fatForbidden are characters vfat (/mnt/onboard) cannot store in names.
const fatForbidden = `<>"|?*\`

// sanitizeRelativePath validates a repo-relative slash path and maps every
// segment to a name that is safe on vfat. It rejects anything that could
// escape the destination folder.
func sanitizeRelativePath(relative string) (string, error) {
	if relative == "" || strings.HasPrefix(relative, "/") {
		return "", i18n.NewError("msg.err_path_absolute", "path", relative)
	}
	segments := strings.Split(relative, "/")
	safe := make([]string, 0, len(segments))
	for _, segment := range segments {
		cleaned, err := sanitizeSegment(segment)
		if err != nil {
			return "", err
		}
		safe = append(safe, cleaned)
	}
	return strings.Join(safe, "/"), nil
}

func sanitizeSegment(segment string) (string, error) {
	if segment == "" || segment == "." || segment == ".." {
		return "", i18n.NewError("msg.err_segment_invalid", "segment", segment)
	}
	if strings.ContainsRune(segment, ':') {
		return "", i18n.NewError("msg.err_segment_colon", "segment", segment)
	}
	// Vietnamese names must be in one Unicode form (NFC) or the same title exists twice on the device.
	var builder strings.Builder
	for _, character := range norm.NFC.String(segment) {
		if character < 0x20 || character == 0x7f {
			return "", i18n.NewError("msg.err_segment_control", "segment", segment)
		}
		if strings.ContainsRune(fatForbidden, character) {
			character = '_'
		}
		builder.WriteRune(character)
	}
	cleaned := strings.TrimRight(builder.String(), ". ")
	if cleaned == "" || len(cleaned) > maxSegmentBytes {
		return "", i18n.NewError("msg.err_name_unusable", "segment", segment)
	}
	return cleaned, nil
}

// localPathFor maps a repo path below basePath to a file under dest.
func localPathFor(dest, basePath, repoPath string) (string, error) {
	relative := repoPath
	if basePath != "" {
		relative = strings.TrimPrefix(repoPath, basePath+"/")
	}
	safe, err := sanitizeRelativePath(relative)
	if err != nil {
		return "", err
	}
	return filepath.Join(dest, filepath.FromSlash(safe)), nil
}
