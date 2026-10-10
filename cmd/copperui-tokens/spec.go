package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"
)

// Units a token can have. Numbers are device pixels unless the unit says otherwise.
const (
	unitPx     = "px"
	unitGrey   = "grey"
	unitMs     = "ms"
	unitCount  = "count"
	unitWeight = "weight"
	unitText   = "text"
	unitFlag   = "flag"
)

const greyStep = 17 // 255 / 15: the panel shows 16 levels

var idPattern = regexp.MustCompile(`^[a-z][A-Za-z0-9]*(\.[a-z][A-Za-z0-9]*)*$`)

// Token is one named constant of the design system.
type Token struct {
	ID    string `json:"id"`
	Value any    `json:"value"`
	Unit  string `json:"unit"`
	Doc   string `json:"doc"`
}

// Group is a titled run of tokens in the generated documentation.
type Group struct {
	Name   string  `json:"name"`
	Tokens []Token `json:"tokens"`
}

// Spec is the parsed token file.
type Spec struct {
	Name    string  `json:"name"`
	Version string  `json:"version"`
	Comment string  `json:"comment"`
	Groups  []Group `json:"groups"`
}

func loadSpec(path string) (*Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read the token file %s: %w", path, err)
	}
	return parseSpec(data)
}

func parseSpec(data []byte) (*Spec, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	var spec Spec
	if err := decoder.Decode(&spec); err != nil {
		return nil, fmt.Errorf("the token file is not valid: %w", err)
	}
	if err := spec.validate(); err != nil {
		return nil, err
	}
	return &spec, nil
}

func (s *Spec) tokens() []Token {
	var all []Token
	for _, group := range s.Groups {
		all = append(all, group.Tokens...)
	}
	return all
}

func (s *Spec) validate() error {
	if s.Version == "" {
		return fmt.Errorf("the token file has no version")
	}
	seenIDs := map[string]bool{}
	seenCpp := map[string]string{}
	for _, token := range s.tokens() {
		if !idPattern.MatchString(token.ID) {
			return fmt.Errorf("token id %q must be dotted camelCase words (for example type.caption.size)", token.ID)
		}
		if seenIDs[token.ID] {
			return fmt.Errorf("token id %q appears twice; keep one", token.ID)
		}
		seenIDs[token.ID] = true
		if other, clash := seenCpp[cppName(token.ID)]; clash {
			return fmt.Errorf("token ids %q and %q give the same C++ name %s; rename one", other, token.ID, cppName(token.ID))
		}
		seenCpp[cppName(token.ID)] = token.ID
		if err := validateValue(token); err != nil {
			return err
		}
	}
	return nil
}

func validateValue(token Token) error {
	switch token.Unit {
	case unitText:
		if _, ok := token.Value.(string); !ok {
			return fmt.Errorf("token %s has unit text, so its value must be a string", token.ID)
		}
	case unitFlag:
		if _, ok := token.Value.(bool); !ok {
			return fmt.Errorf("token %s has unit flag, so its value must be true or false", token.ID)
		}
	case unitPx, unitMs, unitCount, unitWeight, unitGrey:
		number, err := intValue(token)
		if err != nil {
			return err
		}
		if number < 0 {
			return fmt.Errorf("token %s is negative; tokens are never negative", token.ID)
		}
		if token.Unit == unitGrey && (number > 255 || number%greyStep != 0) {
			return fmt.Errorf("token %s = %d is off the 16-level grey grid (multiples of %d up to 255)", token.ID, number, greyStep)
		}
	default:
		return fmt.Errorf("token %s has unknown unit %q", token.ID, token.Unit)
	}
	return nil
}

func intValue(token Token) (int, error) {
	number, ok := token.Value.(json.Number)
	if !ok {
		return 0, fmt.Errorf("token %s (%s) needs a whole number as its value", token.ID, token.Unit)
	}
	value, err := number.Int64()
	if err != nil {
		return 0, fmt.Errorf("token %s (%s) needs a whole number as its value: %w", token.ID, token.Unit, err)
	}
	return int(value), nil
}

// words splits a dotted camelCase id into lower-case words: type.headerTitle.size -> type header title size.
func words(id string) []string {
	var result []string
	for _, segment := range strings.Split(id, ".") {
		start := 0
		for index, char := range segment {
			if index > 0 && unicode.IsUpper(char) {
				result = append(result, strings.ToLower(segment[start:index]))
				start = index
			}
		}
		result = append(result, strings.ToLower(segment[start:]))
	}
	return result
}

// cppName is the constant name: type.headerTitle.size -> TYPE_HEADER_TITLE_SIZE.
func cppName(id string) string {
	return strings.ToUpper(strings.Join(words(id), "_"))
}

// goName is the exported constant name: type.headerTitle.size -> TypeHeaderTitleSize.
func goName(id string) string {
	var name strings.Builder
	for _, word := range words(id) {
		name.WriteString(strings.ToUpper(word[:1]) + word[1:])
	}
	return name.String()
}
