package tweaks

import (
	"fmt"
	"os"
	"strings"
)

// confFile edits Nickel's QSettings ini file line by line, so every line it does not own
// (comments, other sections, odd spacing) survives a write unchanged.
type confFile struct {
	path  string
	lines []string
}

// openConf reads the file; a missing file is an empty one that the first write creates.
func openConf(path string) (*confFile, error) {
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &confFile{path: path}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", path, err)
	}
	text := strings.TrimSuffix(string(content), "\n")
	if text == "" {
		return &confFile{path: path}, nil
	}
	return &confFile{path: path, lines: strings.Split(text, "\n")}, nil
}

// get returns the value of key in section.
func (conf *confFile) get(section, key string) (string, bool) {
	sectionStart, sectionEnd, found := conf.sectionRange(section)
	if !found {
		return "", false
	}
	if index := conf.keyIndex(sectionStart, sectionEnd, key); index >= 0 {
		return conf.valueAt(index), true
	}
	return "", false
}

// set replaces the value of key, or adds the key (and the section) when missing.
func (conf *confFile) set(section, key, value string) {
	line := key + "=" + value
	sectionStart, sectionEnd, found := conf.sectionRange(section)
	if !found {
		conf.appendSection(section, line)
		return
	}
	if index := conf.keyIndex(sectionStart, sectionEnd, key); index >= 0 {
		conf.lines[index] = line
		return
	}
	conf.insertAt(lastContentLine(conf.lines, sectionStart, sectionEnd)+1, line)
}

// remove deletes key from section; a missing key is already removed.
func (conf *confFile) remove(section, key string) {
	sectionStart, sectionEnd, found := conf.sectionRange(section)
	if !found {
		return
	}
	if index := conf.keyIndex(sectionStart, sectionEnd, key); index >= 0 {
		conf.lines = append(conf.lines[:index], conf.lines[index+1:]...)
	}
}

// save writes the file atomically, keeping the permissions Nickel uses.
func (conf *confFile) save() error {
	content := strings.Join(conf.lines, "\n") + "\n"
	return writeFileAtomically(conf.path, []byte(content), 0o644)
}

// sectionRange returns the line index of the section header and the index after its last line.
func (conf *confFile) sectionRange(section string) (start, end int, found bool) {
	header := "[" + section + "]"
	for index, line := range conf.lines {
		if !found && strings.TrimSpace(line) == header {
			start, found = index, true
			continue
		}
		if found && isSectionHeader(line) {
			return start, index, true
		}
	}
	return start, len(conf.lines), found
}

func isSectionHeader(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")
}

func (conf *confFile) keyIndex(sectionStart, sectionEnd int, key string) int {
	for index := sectionStart + 1; index < sectionEnd; index++ {
		name, _, hasEquals := strings.Cut(conf.lines[index], "=")
		if hasEquals && strings.TrimSpace(name) == key {
			return index
		}
	}
	return -1
}

func (conf *confFile) valueAt(index int) string {
	_, value, _ := strings.Cut(conf.lines[index], "=")
	return strings.TrimSpace(value)
}

func (conf *confFile) appendSection(section, line string) {
	if len(conf.lines) > 0 && strings.TrimSpace(conf.lines[len(conf.lines)-1]) != "" {
		conf.lines = append(conf.lines, "")
	}
	conf.lines = append(conf.lines, "["+section+"]", line)
}

func (conf *confFile) insertAt(index int, line string) {
	conf.lines = append(conf.lines, "")
	copy(conf.lines[index+1:], conf.lines[index:])
	conf.lines[index] = line
}

// lastContentLine is the last non-blank line of a section, so a new key stays next to its
// siblings and not after the blank separator.
func lastContentLine(lines []string, sectionStart, sectionEnd int) int {
	for index := sectionEnd - 1; index > sectionStart; index-- {
		if strings.TrimSpace(lines[index]) != "" {
			return index
		}
	}
	return sectionStart
}
