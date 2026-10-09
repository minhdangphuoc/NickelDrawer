package library

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Group kinds.
const (
	KindIdentical = "identical" // every file has the same sha256
	KindProbable  = "probable"  // same normalized title and author, content differs
)

// DuplicateGroup is a set of files that look like the same book.
type DuplicateGroup struct {
	ID    int      `json:"id"`
	Kind  string   `json:"kind"`
	Paths []string `json:"paths"`
}

// HashShared fills SHA256 for every file whose size equals another file's: files of different
// sizes cannot be identical, so most of the library is never read in full.
func HashShared(books []Book, absolute func(relative string) string, logger *Logger) {
	bySize := make(map[int64][]int)
	for index, book := range books {
		bySize[book.Size] = append(bySize[book.Size], index)
	}
	for _, indexes := range bySize {
		if len(indexes) < 2 {
			continue
		}
		for _, index := range indexes {
			sum, err := hashFile(absolute(books[index].Path))
			if err != nil {
				logger.Printf("cannot hash %s: %v", books[index].Path, err)
				continue
			}
			books[index].SHA256 = sum
		}
	}
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// FindDuplicates groups books with the same sha256, or the same normalized title and author.
// Groups are ordered by their first path and numbered from 1.
func FindDuplicates(books []Book) []DuplicateGroup {
	parent := make([]int, len(books))
	for index := range parent {
		parent[index] = index
	}
	var find func(int) int
	find = func(index int) int {
		if parent[index] != index {
			parent[index] = find(parent[index])
		}
		return parent[index]
	}
	joinBy := func(keyOf func(Book) string) {
		firstWithKey := make(map[string]int)
		for index, book := range books {
			key := keyOf(book)
			if key == "" {
				continue
			}
			if first, seen := firstWithKey[key]; seen {
				parent[find(index)] = find(first)
			} else {
				firstWithKey[key] = index
			}
		}
	}
	joinBy(func(book Book) string { return book.SHA256 })
	joinBy(titleAuthorKey)
	return collectGroups(books, find)
}

func collectGroups(books []Book, find func(int) int) []DuplicateGroup {
	members := make(map[int][]int)
	for index := range books {
		root := find(index)
		members[root] = append(members[root], index)
	}
	var groups []DuplicateGroup
	for _, indexes := range members {
		if len(indexes) < 2 {
			continue
		}
		group := DuplicateGroup{Kind: groupKind(books, indexes)}
		for _, index := range indexes {
			group.Paths = append(group.Paths, books[index].Path)
		}
		sort.Strings(group.Paths)
		groups = append(groups, group)
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Paths[0] < groups[j].Paths[0] })
	for index := range groups {
		groups[index].ID = index + 1
	}
	return groups
}

func groupKind(books []Book, indexes []int) string {
	first := books[indexes[0]].SHA256
	for _, index := range indexes {
		if first == "" || books[index].SHA256 != first {
			return KindProbable
		}
	}
	return KindIdentical
}

// titleAuthorKey is "" for anything that is not an EPUB with a title.
func titleAuthorKey(book Book) string {
	if !book.IsEpub() || book.Error != "" || book.Title == "" {
		return ""
	}
	return foldText(book.Title) + "|" + foldText(book.Author)
}

// foldText lower-cases, strips tone marks and drops everything but letters and digits, so
// "Đắc Nhân Tâm" and "dac nhan tam" match.
func foldText(text string) string {
	var folded strings.Builder
	for _, character := range norm.NFD.String(strings.ToLower(text)) {
		switch {
		case character == 'đ':
			folded.WriteByte('d')
		case unicode.Is(unicode.Mn, character):
		case unicode.IsLetter(character) || unicode.IsDigit(character):
			folded.WriteRune(character)
		}
	}
	return folded.String()
}
