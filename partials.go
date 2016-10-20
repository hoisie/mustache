package mustache

import (
	"os"
	"path"
)

// PartialProvider comprises the behaviors required of a struct to be able to provide partials to the mustache rendering
// engine.
type PartialProvider interface {
	// Get accepts the name of a partial and returns the parsed partial, if it could be found; a valid but empty
	// template, if it could not be found; or nil and error if an error occurred (other than an inability to find
	// the partial).
	Get(name string) (*Template, error)
}

// FileProvider implements the PartialProvider interface by providing partials drawn from a filesystem. When a partial
// named `NAME`  is requested, FileProvider searches each listed path for a file named as `NAME` followed by any of the
// listed extensions. The default for `Paths` is to search the current working directory. The default for `Extensions`
// is to examine, in order, no extension; then ".mustache"; then ".stache".
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

var _ PartialProvider = (*FileProvider)(nil)

// StaticProvider implements the PartialProvider interface by providing partials drawn from a map, which maps partial
// name to template contents.
type StaticProvider struct {
	Partials map[string]string
}

func (sp *StaticProvider) Get(name string) (*Template, error) {
	if sp.Partials != nil {
		if data, ok := sp.Partials[name]; ok {
			return ParseStringPartials(data, sp)
		}
	}

	return ParseString("")
}

var _ PartialProvider = (*StaticProvider)(nil)
