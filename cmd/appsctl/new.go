package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"koboapps/templates"
)

const templateSuffix = ".tmpl"

var appIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,30}$`)

type newAppValues struct {
	ID     string
	Name   string
	WithUI bool
}

func runNew(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("new", flag.ContinueOnError)
	flags.SetOutput(stderr)
	name := flags.String("name", "", "app name shown in the menu (default: derived from the id)")
	withUI := flags.Bool("with-ui", false, "also create ui.json and a state action")
	parentDir := flags.String("dir", ".", "folder that will contain the new app folder")
	positional, err := parseNewArgs(flags, args)
	if err != nil || len(positional) != 1 {
		fmt.Fprintln(stderr, "usage: appsctl new <id> [--name \"My App\"] [--with-ui] [--dir DIR]")
		return exitFailure
	}
	values := newAppValues{ID: positional[0], Name: *name, WithUI: *withUI}
	if values.Name == "" {
		values.Name = displayNameFromID(values.ID)
	}
	appDir := filepath.Join(*parentDir, values.ID)
	if err := scaffoldApp(appDir, values); err != nil {
		fmt.Fprintln(stderr, "appsctl new:", err)
		return exitFailure
	}
	fmt.Fprintf(stdout, "created %s\n", appDir)
	if reportDirs([]string{appDir}, stdout) != exitOK {
		fmt.Fprintln(stderr, "appsctl new: the generated app does not validate; this is a bug in templates/app")
		return exitFailure
	}
	printNewAppNextSteps(stdout, appDir, values.WithUI)
	return exitOK
}

// parseNewArgs accepts flags before or after the id, as `appsctl new hello --with-ui` reads naturally.
func parseNewArgs(flags *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for len(args) > 0 {
		if err := flags.Parse(args); err != nil {
			return nil, err
		}
		args = flags.Args()
		if len(args) > 0 {
			positional = append(positional, args[0])
			args = args[1:]
		}
	}
	return positional, nil
}

func displayNameFromID(id string) string {
	words := strings.Split(id, "-")
	for i, word := range words {
		if word != "" {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

func validateNewAppValues(values newAppValues) error {
	if !appIDPattern.MatchString(values.ID) {
		return fmt.Errorf("invalid id %q: use 2 to 31 lowercase letters, digits or '-', starting with a letter or digit", values.ID)
	}
	nameLength := len([]rune(values.Name))
	if nameLength < 1 || nameLength > 24 || strings.Contains(values.Name, ":") {
		return fmt.Errorf("invalid name %q: use 1 to 24 characters without ':'", values.Name)
	}
	return nil
}

func scaffoldApp(appDir string, values newAppValues) error {
	if err := validateNewAppValues(values); err != nil {
		return err
	}
	if _, err := os.Stat(appDir); err == nil {
		return fmt.Errorf("%s already exists; pick another id or --dir, nothing was changed", appDir)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("cannot check %s: %w", appDir, err)
	}
	if err := writeTemplateTree(appDir, values); err != nil {
		os.RemoveAll(appDir)
		return err
	}
	return nil
}

func writeTemplateTree(appDir string, values newAppValues) error {
	return fs.WalkDir(templates.App, "app", func(source string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative := strings.TrimPrefix(strings.TrimPrefix(source, "app"), "/")
		if shouldSkipTemplate(relative, values) {
			return nil
		}
		target := filepath.Join(appDir, strings.TrimSuffix(relative, templateSuffix))
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return renderTemplateFile(source, target, values)
	})
}

// shouldSkipTemplate leaves out the optional ui.json and state script unless --with-ui is set.
func shouldSkipTemplate(relative string, values newAppValues) bool {
	if values.WithUI {
		return false
	}
	return relative == "ui.json.tmpl" || relative == "bin/state.sh.tmpl"
}

func renderTemplateFile(source, target string, values newAppValues) error {
	raw, err := fs.ReadFile(templates.App, source)
	if err != nil {
		return fmt.Errorf("cannot read template %s: %w", source, err)
	}
	parsed, err := template.New(path.Base(source)).Funcs(template.FuncMap{"json": jsonString}).Parse(string(raw))
	if err != nil {
		return fmt.Errorf("template %s is broken: %w", source, err)
	}
	var rendered strings.Builder
	if err := parsed.Execute(&rendered, values); err != nil {
		return fmt.Errorf("cannot render %s: %w", source, err)
	}
	mode := fs.FileMode(0o644)
	if strings.HasPrefix(source, "app/bin/") {
		mode = 0o755
	}
	return os.WriteFile(target, []byte(rendered.String()), mode)
}

func jsonString(text string) (string, error) {
	encoded, err := json.Marshal(text)
	return string(encoded), err
}

func printNewAppNextSteps(out io.Writer, appDir string, hasUI bool) {
	fmt.Fprintln(out, "\nNext steps:")
	step := 1
	printStep := func(format string, args ...any) {
		fmt.Fprintf(out, "  %d. ", step)
		fmt.Fprintf(out, format+"\n", args...)
		step++
	}
	printStep("Edit %s and %s", filepath.Join(appDir, "bin", "run.sh"), filepath.Join(appDir, "app.json"))
	printStep("Validate:  appsctl validate %s", appDir)
	if hasUI {
		printStep("Preview:   ndpreview serve %s", appDir)
	}
	printStep("Install:   copy it to /mnt/onboard/.adds/apps/%s/ (see docs/guide/developer-guide.md, \"Install on a Kobo\")", filepath.Base(appDir))
}
