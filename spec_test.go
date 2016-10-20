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
		"Standalone":                       false,
		"Indented Standalone":              false,
		"Standalone Line Endings":          false,
		"Standalone Without Previous Line": false,
		"Standalone Without Newline":       false,
		"Multiline Standalone":             false,
		"Indented Multiline Standalone":    false,
		"Indented Inline":                  true,
		"Surrounding Whitespace":           true,
	},
	"delimiters.json": map[string]bool{
		"Pair Behavior":                    true,
		"Special Characters":               true,
		"Sections":                         false,
		"Inverted Sections":                false,
		"Partial Inheritence":              false,
		"Post-Partial Behavior":            true,
		"Outlying Whitespace (Inline)":     true,
		"Standalone Tag":                   false,
		"Indented Standalone Tag":          false,
		"Pair with Padding":                true,
		"Surrounding Whitespace":           true,
		"Standalone Line Endings":          false,
		"Standalone Without Previous Line": false,
		"Standalone Without Newline":       false,
	},
	"interpolation.json": map[string]bool{
		"No Interpolation":                             true,
		"Basic Interpolation":                          true,
		"HTML Escaping":                                false,
		"Triple Mustache":                              true,
		"Ampersand":                                    false,
		"Basic Integer Interpolation":                  true,
		"Triple Mustache Integer Interpolation":        true,
		"Ampersand Integer Interpolation":              false,
		"Basic Decimal Interpolation":                  true,
		"Triple Mustache Decimal Interpolation":        true,
		"Ampersand Decimal Interpolation":              false,
		"Basic Context Miss Interpolation":             true,
		"Triple Mustache Context Miss Interpolation":   true,
		"Ampersand Context Miss Interpolation":         true,
		"Dotted Names - Basic Interpolation":           true,
		"Dotted Names - Triple Mustache Interpolation": true,
		"Dotted Names - Ampersand Interpolation":       false,
		"Dotted Names - Arbitrary Depth":               true,
		"Dotted Names - Broken Chains":                 true,
		"Dotted Names - Broken Chain Resolution":       true,
		"Dotted Names - Initial Resolution":            true,
		"Interpolation - Surrounding Whitespace":       true,
		"Triple Mustache - Surrounding Whitespace":     true,
		"Ampersand - Surrounding Whitespace":           false,
		"Interpolation - Standalone":                   true,
		"Triple Mustache - Standalone":                 true,
		"Ampersand - Standalone":                       false,
		"Interpolation With Padding":                   true,
		"Triple Mustache With Padding":                 false,
		"Ampersand With Padding":                       false,
	},
	"inverted.json": map[string]bool{
		"Falsey":                           true,
		"Truthy":                           true,
		"Context":                          true,
		"List":                             true,
		"Empty List":                       true,
		"Doubled":                          false,
		"Nested (Falsey)":                  true,
		"Nested (Truthy)":                  true,
		"Context Misses":                   true,
		"Dotted Names - Truthy":            true,
		"Dotted Names - Falsey":            true,
		"Internal Whitespace":              true,
		"Indented Inline Sections":         true,
		"Standalone Lines":                 false,
		"Standalone Indented Lines":        false,
		"Padding":                          true,
		"Dotted Names - Broken Chains":     true,
		"Surrounding Whitespace":           true,
		"Standalone Line Endings":          false,
		"Standalone Without Previous Line": false,
		"Standalone Without Newline":       false,
	},
	"partials.json": map[string]bool{
		"Basic Behavior":                   true,
		"Failed Lookup":                    true,
		"Context":                          true,
		"Recursion":                        true,
		"Surrounding Whitespace":           true,
		"Inline Indentation":               true,
		"Standalone Line Endings":          false,
		"Standalone Without Previous Line": false,
		"Standalone Without Newline":       false,
		"Standalone Indentation":           false,
		"Padding Whitespace":               true,
	},
	"sections.json": map[string]bool{
		"Truthy":                 true,
		"Falsey":                 true,
		"Context":                true,
		"Deeply Nested Contexts": false,
		"List":                             true,
		"Empty List":                       true,
		"Doubled":                          false,
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
		"Standalone Lines":                 false,
		"Indented Standalone Lines":        false,
		"Standalone Line Endings":          false,
		"Standalone Without Previous Line": false,
		"Standalone Without Newline":       false,
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
