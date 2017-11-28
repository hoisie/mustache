package mustache

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

var enabledTests = map[string]map[string]bool{
	"comments.json": map[string]bool{
		"Inline":                           true,
		"Multiline":                        true,
		"Standalone":                       true,
		"Indented Standalone":              true,
		"Standalone Line Endings":          true,
		"Standalone Without Previous Line": true,
		"Standalone Without Newline":       true,
		"Multiline Standalone":             true,
		"Indented Multiline Standalone":    true,
		"Indented Inline":                  true,
		"Surrounding Whitespace":           true,
	},
	"delimiters.json": map[string]bool{
		"Pair Behavior":                    true,
		"Special Characters":               true,
		"Sections":                         true,
		"Inverted Sections":                true,
		"Partial Inheritence":              true,
		"Post-Partial Behavior":            true,
		"Outlying Whitespace (Inline)":     true,
		"Standalone Tag":                   true,
		"Indented Standalone Tag":          true,
		"Pair with Padding":                true,
		"Surrounding Whitespace":           true,
		"Standalone Line Endings":          true,
		"Standalone Without Previous Line": true,
		"Standalone Without Newline":       true,
	},
	"interpolation.json": map[string]bool{
		"No Interpolation":    true,
		"Basic Interpolation": true,
		// disabled b/c Go uses "&#34;" in place of "&quot;"
		// both are valid escapings, and we validate the behavior in mustache_test.go
		"HTML Escaping":                                false,
		"Triple Mustache":                              true,
		"Ampersand":                                    true,
		"Basic Integer Interpolation":                  true,
		"Triple Mustache Integer Interpolation":        true,
		"Ampersand Integer Interpolation":              true,
		"Basic Decimal Interpolation":                  true,
		"Triple Mustache Decimal Interpolation":        true,
		"Ampersand Decimal Interpolation":              true,
		"Basic Context Miss Interpolation":             true,
		"Triple Mustache Context Miss Interpolation":   true,
		"Ampersand Context Miss Interpolation":         true,
		"Dotted Names - Basic Interpolation":           true,
		"Dotted Names - Triple Mustache Interpolation": true,
		"Dotted Names - Ampersand Interpolation":       true,
		"Dotted Names - Arbitrary Depth":               true,
		"Dotted Names - Broken Chains":                 true,
		"Dotted Names - Broken Chain Resolution":       true,
		"Dotted Names - Initial Resolution":            true,
		"Interpolation - Surrounding Whitespace":       true,
		"Triple Mustache - Surrounding Whitespace":     true,
		"Ampersand - Surrounding Whitespace":           true,
		"Interpolation - Standalone":                   true,
		"Triple Mustache - Standalone":                 true,
		"Ampersand - Standalone":                       true,
		"Interpolation With Padding":                   true,
		"Triple Mustache With Padding":                 true,
		"Ampersand With Padding":                       true,
	},
	"inverted.json": map[string]bool{
		"Falsey":                           true,
		"Truthy":                           true,
		"Context":                          true,
		"List":                             true,
		"Empty List":                       true,
		"Doubled":                          true,
		"Nested (Falsey)":                  true,
		"Nested (Truthy)":                  true,
		"Context Misses":                   true,
		"Dotted Names - Truthy":            true,
		"Dotted Names - Falsey":            true,
		"Internal Whitespace":              true,
		"Indented Inline Sections":         true,
		"Standalone Lines":                 true,
		"Standalone Indented Lines":        true,
		"Padding":                          true,
		"Dotted Names - Broken Chains":     true,
		"Surrounding Whitespace":           true,
		"Standalone Line Endings":          true,
		"Standalone Without Previous Line": true,
		"Standalone Without Newline":       true,
	},
	"partials.json": map[string]bool{
		"Basic Behavior":                   true,
		"Failed Lookup":                    true,
		"Context":                          true,
		"Recursion":                        true,
		"Surrounding Whitespace":           true,
		"Inline Indentation":               true,
		"Standalone Line Endings":          true,
		"Standalone Without Previous Line": true,
		"Standalone Without Newline":       true,
		"Standalone Indentation":           true,
		"Padding Whitespace":               true,
	},
	"sections.json": map[string]bool{
		"Truthy":                 true,
		"Falsey":                 true,
		"Context":                true,
		"Deeply Nested Contexts": true,
		"List":                             true,
		"Empty List":                       true,
		"Doubled":                          true,
		"Nested (Truthy)":                  true,
		"Nested (Falsey)":                  true,
		"Context Misses":                   true,
		"Implicit Iterator - String":       true,
		"Implicit Iterator - Integer":      true,
		"Implicit Iterator - Decimal":      true,
		"Implicit Iterator - Array":        true,
		"Dotted Names - Truthy":            true,
		"Dotted Names - Falsey":            true,
		"Dotted Names - Broken Chains":     true,
		"Surrounding Whitespace":           true,
		"Internal Whitespace":              true,
		"Indented Inline Sections":         true,
		"Standalone Lines":                 true,
		"Indented Standalone Lines":        true,
		"Standalone Line Endings":          true,
		"Standalone Without Previous Line": true,
		"Standalone Without Newline":       true,
		"Padding":                          true,
	},
	"~lambdas.json": nil, // not implemented
}

type specTest struct {
	Name        string            `json:"name"`
	Data        interface{}       `json:"data"`
	Expected    string            `json:"expected"`
	Template    string            `json:"template"`
	Description string            `json:"desc"`
	Partials    map[string]string `json:"partials"`
}

type specTestSuite struct {
	Tests []specTest `json:"tests"`
}

func TestSpec(t *testing.T) {
	root := filepath.Join(os.Getenv("PWD"), "spec", "specs")
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("Could not find the specs folder at %s, ensure the submodule exists by running 'git submodule update --init'", root)
		}
		t.Fatal(err)
	}

	paths, err := filepath.Glob(root + "/*.json")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(paths)

	for _, path := range paths {
		_, file := filepath.Split(path)
		enabled, ok := enabledTests[file]
		if !ok {
			t.Errorf("Unexpected file %s, consider adding to enabledFiles", file)
			continue
		}
		if enabled == nil {
			continue
		}
		b, err := ioutil.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var suite specTestSuite
		err = json.Unmarshal(b, &suite)
		if err != nil {
			t.Fatal(err)
		}
		for _, test := range suite.Tests {
			runTest(t, file, &test)
		}
	}
}

func runTest(t *testing.T, file string, test *specTest) {
	enabled, ok := enabledTests[file][test.Name]
	if !ok {
		t.Errorf("[%s %s]: Unexpected test, add to enabledTests", file, test.Name)
	}
	if !enabled {
		t.Logf("[%s %s]: Skipped", file, test.Name)
		return
	}

	var out string
	var err error
	if len(test.Partials) > 0 {
		out, err = RenderPartials(test.Template, &StaticProvider{test.Partials}, test.Data)
	} else {
		out, err = Render(test.Template, test.Data)
	}
	if err != nil {
		t.Errorf("[%s %s]: %s", file, test.Name, err.Error())
		return
	}
	if out != test.Expected {
		t.Errorf("[%s %s]: Expected %q, got %q", file, test.Name, test.Expected, out)
		return
	}

	t.Logf("[%s %s]: Passed", file, test.Name)
}
