package mustache

import (
	"os"
	"path"
)

type PartialProvider interface {
	Get(name string) (*Template, error)
}

type FileProvider struct {
	Paths      []string
	Extensions []string
}

func (fp *FileProvider) Get(name string) (*Template, error) {
	var filename string

	var paths []string
	if fp.Paths != nil {
		paths = fp.Paths
	} else {
		paths = []string{""}
	}

	var exts []string
	if fp.Extensions != nil {
		exts = fp.Extensions
	} else {
		exts = []string{"", ".mustache", ".stache"}
	}

	for _, p := range paths {
		for _, e := range exts {
			name := path.Join(p, name+e)
			f, err := os.Open(name)
			if err == nil {
				filename = name
				f.Close()
				break
			}
		}
	}

	if filename == "" {
		return ParseString("")
	}

	return ParseFile(filename)
}

type StaticProvider map[string]string

func (sp StaticProvider) Get(name string) (*Template, error) {
	if data, ok := sp[name]; ok {
		return ParseStringPartials(data, sp)
	}

	return ParseString("")
}
