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

	for _, p := range fp.Paths {
		for _, e := range fp.Extensions {
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
