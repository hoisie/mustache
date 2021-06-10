package mustache

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

var disabledTests = map[string]map[string]struct{}{
	"interpolation.json": {
		// disabled b/c Go uses "&#34;" in place of "&quot;"
		// both are valid escapings, and we validate the behavior in mustache_test.go
		"HTML Escaping": struct{}{},
		// Newly added spec tests which aren't currently passing:
		"Basic Null Interpolation":           struct{}{},
		"Triple Mustache Null Interpolation": struct{}{},
		"Ampersand Null Interpolation":       struct{}{},
		"Implicit Iterators - HTML Escaping": struct{}{},
	},
	// To be fixed by https://github.com/cbroglie/mustache/pull/55
	"sections.json": {
		"Variable test":          struct{}{},
		"Deeply Nested Contexts": struct{}{},
	},
	"~lambdas.json":     {}, // not implemented
	"~inheritance.json": {}, // not implemented
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
	disabled, ok := disabledTests[file]
	if ok {
		// Can disable a single test or the entire file.
		if _, ok := disabled[test.Name]; ok || len(disabled) == 0 {
			t.Logf("[%s %s]: Skipped", file, test.Name)
			return
		}
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
