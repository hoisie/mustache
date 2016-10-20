package mustache

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
)

var (
	// AllowMissingVariables defines the behavior for a variable "miss." If it
	// is true (the default), an empty string is emitted. If it is false, an error
	// is generated instead.
	AllowMissingVariables = true
)

// A TagType represents the specific type of mustache tag that a Tag
// represents. The zero TagType is not a valid type.
type TagType uint

// Defines representing the possible Tag types
const (
	Invalid TagType = iota
	Variable
	Section
	InvertedSection
	Partial
)

func (t TagType) String() string {
	if int(t) < len(tagNames) {
		return tagNames[t]
	}
	return "type" + strconv.Itoa(int(t))
}

var tagNames = []string{
	Invalid:         "Invalid",
	Variable:        "Variable",
	Section:         "Section",
	InvertedSection: "InvertedSection",
	Partial:         "Partial",
}

// Tag represents the different mustache tag types.
//
// Not all methods apply to all kinds of tags. Restrictions, if any, are noted
// in the documentation for each method. Use the Type method to find out the
// type of tag before calling type-specific methods. Calling a method
// inappropriate to the type of tag causes a run time panic.
type Tag interface {
	// Type returns the type of the tag.
	Type() TagType
	// Name returns the name of the tag.
	Name() string
	// Tags returns any child tags. It panics for tag types which cannot contain
	// child tags (i.e. variable tags).
	Tags() []Tag
}

type textElement struct {
	text []byte
}

type varElement struct {
	name string
	raw  bool
}

type sectionElement struct {
	name      string
	inverted  bool
	startline int
	elems     []interface{}
}

type partialElement struct {
	name string
	prov PartialProvider
}

// Template represents a compilde mustache template
type Template struct {
	data    string
	otag    string
	ctag    string
	p       int
	curline int
	dir     string
	elems   []interface{}
	partial PartialProvider
}

type parseError struct {
	line    int
	message string
}

// Tags returns the mustache tags for the given template
func (tmpl *Template) Tags() []Tag {
	return extractTags(tmpl.elems)
}

func extractTags(elems []interface{}) []Tag {
	tags := make([]Tag, 0, len(elems))
	for _, elem := range elems {
		switch elem := elem.(type) {
		case *varElement:
			tags = append(tags, elem)
		case *sectionElement:
			tags = append(tags, elem)
		case *partialElement:
			tags = append(tags, elem)
		}
	}
	return tags
}

func (e *varElement) Type() TagType {
	return Variable
}

func (e *varElement) Name() string {
	return e.name
}

func (e *varElement) Tags() []Tag {
	panic("mustache: Tags on Variable type")
}

func (e *sectionElement) Type() TagType {
	if e.inverted {
		return InvertedSection
	}
	return Section
}

func (e *sectionElement) Name() string {
	return e.name
}

func (e *sectionElement) Tags() []Tag {
	return extractTags(e.elems)
}

func (e *partialElement) Type() TagType {
	return Partial
}

func (e *partialElement) Name() string {
	return e.name
}

func (e *partialElement) Tags() []Tag {
	return nil
}

func (p parseError) Error() string {
	return fmt.Sprintf("line %d: %s", p.line, p.message)
}

func (tmpl *Template) readString(s string) (string, error) {
	newlines := 0
	for i := tmpl.p; ; i++ {
		//are we at the end of the string?
		if i+len(s) > len(tmpl.data) {
			return tmpl.data[tmpl.p:], io.EOF
		}

		if tmpl.data[i] == '\n' {
			newlines++
		}

		if tmpl.data[i] != s[0] {
			continue
		}

		match := true
		for j := 1; j < len(s); j++ {
			if s[j] != tmpl.data[i+j] {
				match = false
				break
			}
		}

		if match {
			e := i + len(s)
			text := tmpl.data[tmpl.p:e]
			tmpl.p = e

			tmpl.curline += newlines
			return text, nil
		}
	}
}

func (tmpl *Template) parsePartial(name string) (*partialElement, error) {
	var prov PartialProvider
	if tmpl.partial == nil {
		prov = &FileProvider{
			Paths: []string{tmpl.dir, " "},
		}
	} else {
		prov = tmpl.partial
	}

	return &partialElement{
		name: name,
		prov: prov,
	}, nil
}

func (tmpl *Template) parseSection(section *sectionElement) error {
	for {
		text, err := tmpl.readString(tmpl.otag)

		if err == io.EOF {
			return parseError{section.startline, "Section " + section.name + " has no closing tag"}
		}

		// put text into an item
		text = text[0 : len(text)-len(tmpl.otag)]
		section.elems = append(section.elems, &textElement{[]byte(text)})
		if tmpl.p < len(tmpl.data) && tmpl.data[tmpl.p] == '{' {
			text, err = tmpl.readString("}" + tmpl.ctag)
		} else {
			text, err = tmpl.readString(tmpl.ctag)
		}

		if err == io.EOF {
			//put the remaining text in a block
			return parseError{tmpl.curline, "unmatched open tag"}
		}

		//trim the close tag off the text
		tag := strings.TrimSpace(text[0 : len(text)-len(tmpl.ctag)])

		if len(tag) == 0 {
			return parseError{tmpl.curline, "empty tag"}
		}
		switch tag[0] {
		case '!':
			//ignore comment
			break
		case '#', '^':
			name := strings.TrimSpace(tag[1:])

			//ignore the newline when a section starts
			if len(tmpl.data) > tmpl.p && tmpl.data[tmpl.p] == '\n' {
				tmpl.p++
			} else if len(tmpl.data) > tmpl.p+1 && tmpl.data[tmpl.p] == '\r' && tmpl.data[tmpl.p+1] == '\n' {
				tmpl.p += 2
			}

			se := sectionElement{name, tag[0] == '^', tmpl.curline, []interface{}{}}
			err := tmpl.parseSection(&se)
			if err != nil {
				return err
			}
			section.elems = append(section.elems, &se)
		case '/':
			name := strings.TrimSpace(tag[1:])
			if name != section.name {
				return parseError{tmpl.curline, "interleaved closing tag: " + name}
			}
			return nil
		case '>':
			name := strings.TrimSpace(tag[1:])
			partial, err := tmpl.parsePartial(name)
			if err != nil {
				return err
			}
			section.elems = append(section.elems, partial)
		case '=':
			if tag[len(tag)-1] != '=' {
				return parseError{tmpl.curline, "Invalid meta tag"}
			}
			tag = strings.TrimSpace(tag[1 : len(tag)-1])
			newtags := strings.SplitN(tag, " ", 2)
			if len(newtags) == 2 {
				tmpl.otag = newtags[0]
				tmpl.ctag = newtags[1]
			}
		case '{':
			if tag[len(tag)-1] == '}' {
				//use a raw tag
				section.elems = append(section.elems, &varElement{tag[1 : len(tag)-1], true})
			}
		default:
			section.elems = append(section.elems, &varElement{tag, false})
		}
	}
}

func (tmpl *Template) parse() error {
	for {
		text, err := tmpl.readString(tmpl.otag)
		if err == io.EOF {
			//put the remaining text in a block
			tmpl.elems = append(tmpl.elems, &textElement{[]byte(text)})
			return nil
		}

		// put text into an item
		text = text[0 : len(text)-len(tmpl.otag)]
		tmpl.elems = append(tmpl.elems, &textElement{[]byte(text)})

		if tmpl.p < len(tmpl.data) && tmpl.data[tmpl.p] == '{' {
			text, err = tmpl.readString("}" + tmpl.ctag)
		} else {
			text, err = tmpl.readString(tmpl.ctag)
		}

		if err == io.EOF {
			//put the remaining text in a block
			return parseError{tmpl.curline, "unmatched open tag"}
		}

		//trim the close tag off the text
		tag := strings.TrimSpace(text[0 : len(text)-len(tmpl.ctag)])
		if len(tag) == 0 {
			return parseError{tmpl.curline, "empty tag"}
		}
		switch tag[0] {
		case '!':
			//ignore comment
			break
		case '#', '^':
			name := strings.TrimSpace(tag[1:])

			if len(tmpl.data) > tmpl.p && tmpl.data[tmpl.p] == '\n' {
				tmpl.p++
			} else if len(tmpl.data) > tmpl.p+1 && tmpl.data[tmpl.p] == '\r' && tmpl.data[tmpl.p+1] == '\n' {
				tmpl.p += 2
			}

			se := sectionElement{name, tag[0] == '^', tmpl.curline, []interface{}{}}
			err := tmpl.parseSection(&se)
			if err != nil {
				return err
			}
			tmpl.elems = append(tmpl.elems, &se)
		case '/':
			return parseError{tmpl.curline, "unmatched close tag"}
		case '>':
			name := strings.TrimSpace(tag[1:])
			partial, err := tmpl.parsePartial(name)
			if err != nil {
				return err
			}
			tmpl.elems = append(tmpl.elems, partial)
		case '=':
			if tag[len(tag)-1] != '=' {
				return parseError{tmpl.curline, "Invalid meta tag"}
			}
			tag = strings.TrimSpace(tag[1 : len(tag)-1])
			newtags := strings.SplitN(tag, " ", 2)
			if len(newtags) == 2 {
				tmpl.otag = newtags[0]
				tmpl.ctag = newtags[1]
			}
		case '{':
			//use a raw tag
			if tag[len(tag)-1] == '}' {
				tmpl.elems = append(tmpl.elems, &varElement{tag[1 : len(tag)-1], true})
			}
		default:
			tmpl.elems = append(tmpl.elems, &varElement{tag, false})
		}
	}
}

// Evaluate interfaces and pointers looking for a value that can look up the name, via a
// struct field, method, or map key, and return the result of the lookup.
func lookup(contextChain []interface{}, name string, allowMissing bool) (reflect.Value, error) {
	// dot notation
	if name != "." && strings.Contains(name, ".") {
		parts := strings.SplitN(name, ".", 2)

		v, err := lookup(contextChain, parts[0], allowMissing)
		if err != nil {
			return v, err
		}
		return lookup([]interface{}{v}, parts[1], allowMissing)
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic while looking up %q: %s\n", name, r)
		}
	}()

Outer:
	for _, ctx := range contextChain {
		v := ctx.(reflect.Value)
		for v.IsValid() {
			typ := v.Type()
			if n := v.Type().NumMethod(); n > 0 {
				for i := 0; i < n; i++ {
					m := typ.Method(i)
					mtyp := m.Type
					if m.Name == name && mtyp.NumIn() == 1 {
						return v.Method(i).Call(nil)[0], nil
					}
				}
			}
			if name == "." {
				return v, nil
			}
			switch av := v; av.Kind() {
			case reflect.Ptr:
				v = av.Elem()
			case reflect.Interface:
				v = av.Elem()
			case reflect.Struct:
				ret := av.FieldByName(name)
				if ret.IsValid() {
					return ret, nil
				}
				continue Outer
			case reflect.Map:
				ret := av.MapIndex(reflect.ValueOf(name))
				if ret.IsValid() {
					return ret, nil
				}
				continue Outer
			default:
				continue Outer
			}
		}
	}
	if allowMissing {
		return reflect.Value{}, nil
	}
	return reflect.Value{}, fmt.Errorf("Missing variable %q", name)
}

func isEmpty(v reflect.Value) bool {
	if !v.IsValid() || v.Interface() == nil {
		return true
	}

	valueInd := indirect(v)
	if !valueInd.IsValid() {
		return true
	}
	switch val := valueInd; val.Kind() {
	case reflect.Bool:
		return !val.Bool()
	case reflect.Slice:
		return val.Len() == 0
	case reflect.String:
		return len(strings.TrimSpace(val.String())) == 0
	}

	return false
}

func indirect(v reflect.Value) reflect.Value {
loop:
	for v.IsValid() {
		switch av := v; av.Kind() {
		case reflect.Ptr:
			v = av.Elem()
		case reflect.Interface:
			v = av.Elem()
		default:
			break loop
		}
	}
	return v
}

func renderSection(section *sectionElement, contextChain []interface{}, buf io.Writer) error {
	value, err := lookup(contextChain, section.name, true)
	if err != nil {
		return err
	}
	var context = contextChain[len(contextChain)-1].(reflect.Value)
	var contexts = []interface{}{}
	// if the value is nil, check if it's an inverted section
	isEmpty := isEmpty(value)
	if isEmpty && !section.inverted || !isEmpty && section.inverted {
		return nil
	} else if !section.inverted {
		valueInd := indirect(value)
		switch val := valueInd; val.Kind() {
		case reflect.Slice:
			for i := 0; i < val.Len(); i++ {
				contexts = append(contexts, val.Index(i))
			}
		case reflect.Array:
			for i := 0; i < val.Len(); i++ {
				contexts = append(contexts, val.Index(i))
			}
		case reflect.Map, reflect.Struct:
			contexts = append(contexts, value)
		default:
			contexts = append(contexts, context)
		}
	} else if section.inverted {
		contexts = append(contexts, context)
	}

	chain2 := make([]interface{}, len(contextChain)+1)
	copy(chain2[1:], contextChain)
	//by default we execute the section
	for _, ctx := range contexts {
		chain2[0] = ctx
		for _, elem := range section.elems {
			renderElement(elem, chain2, buf)
		}
	}
	return nil
}

func renderElement(element interface{}, contextChain []interface{}, buf io.Writer) error {
	switch elem := element.(type) {
	case *textElement:
		buf.Write(elem.text)
	case *varElement:
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Panic while looking up %q: %s\n", elem.name, r)
			}
		}()
		val, err := lookup(contextChain, elem.name, AllowMissingVariables)
		if err != nil {
			return err
		}

		if val.IsValid() {
			if elem.raw {
				fmt.Fprint(buf, val.Interface())
			} else {
				s := fmt.Sprint(val.Interface())
				template.HTMLEscape(buf, []byte(s))
			}
		}
	case *sectionElement:
		if err := renderSection(elem, contextChain, buf); err != nil {
			return err
		}
	case *partialElement:
		partial, err := elem.prov.Get(elem.name)
		if err != nil {
			return err
		}
		if err := partial.renderTemplate(contextChain, buf); err != nil {
			return err
		}
	}
	return nil
}

func (tmpl *Template) renderTemplate(contextChain []interface{}, buf io.Writer) error {
	for _, elem := range tmpl.elems {
		if err := renderElement(elem, contextChain, buf); err != nil {
			return err
		}
	}
	return nil
}

// FRender uses the given data source - generally a map or struct - to
// render the compiled template to an io.Writer.
func (tmpl *Template) FRender(out io.Writer, context ...interface{}) error {
	var contextChain []interface{}
	for _, c := range context {
		val := reflect.ValueOf(c)
		contextChain = append(contextChain, val)
	}
	return tmpl.renderTemplate(contextChain, out)
}

// Render uses the given data source - generally a map or struct - to render
// the compiled template and return the output.
func (tmpl *Template) Render(context ...interface{}) (string, error) {
	var buf bytes.Buffer
	err := tmpl.FRender(&buf, context...)
	return buf.String(), err
}

// RenderInLayout uses the given data source - generally a map or struct - to
// render the compiled template and layout "wrapper" template and return the
// output.
func (tmpl *Template) RenderInLayout(layout *Template, context ...interface{}) (string, error) {
	var buf bytes.Buffer
	err := tmpl.FRenderInLayout(&buf, layout, context...)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// FRenderInLayout uses the given data source - generally a map or
// struct - to render the compiled templated a loayout "wrapper"
// template to an io.Writer.
func (tmpl *Template) FRenderInLayout(out io.Writer, layout *Template, context ...interface{}) error {
	content, err := tmpl.Render(context...)
	if err != nil {
		return err
	}
	allContext := make([]interface{}, len(context)+1)
	copy(allContext[1:], context)
	allContext[0] = map[string]string{"content": content}
	return layout.FRender(out, allContext...)
}

// ParseString compiles a mustache template string. The resulting output can
// be used to efficiently render the template multiple times with different data
// sources.
func ParseString(data string) (*Template, error) {
	return ParseStringPartials(data, nil)
}

// ParseStringPartials compiles a mustache template string, retrieving any
// required partials from the given provider. The resulting output can be used
// to efficiently render the template multiple times with different data
// sources.
func ParseStringPartials(data string, partials PartialProvider) (*Template, error) {
	cwd := os.Getenv("CWD")
	tmpl := Template{data, "{{", "}}", 0, 1, cwd, []interface{}{}, partials}
	err := tmpl.parse()

	if err != nil {
		return nil, err
	}

	return &tmpl, err
}

// ParseFile loads a mustache template string from a file and compiles it. The
// resulting output can be used to efficiently render the template multiple
// times with different data sources.
func ParseFile(filename string) (*Template, error) {
	return ParseFilePartials(filename, nil)
}

// ParseFilePartials loads a mustache template string from a file, retrieving any
// required partials from the given provider, and compiles it. The resulting
// output can be used to efficiently render the template multiple times with
// different data sources.
func ParseFilePartials(filename string, partials PartialProvider) (*Template, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	dirname, _ := path.Split(filename)

	tmpl := Template{string(data), "{{", "}}", 0, 1, dirname, []interface{}{}, partials}
	err = tmpl.parse()

	if err != nil {
		return nil, err
	}

	return &tmpl, nil
}

// Render compiles a mustache template string and uses the the given data source
// - generally a map or struct - to render the template and return the output.
func Render(data string, context ...interface{}) (string, error) {
	return RenderPartials(data, nil, context...)
}

// RenderPartials compiles a mustache template string and uses the the given partial
// provider and data source - generally a map or struct - to render the template
// and return the output.
func RenderPartials(data string, partials PartialProvider, context ...interface{}) (string, error) {
	var tmpl *Template
	var err error
	if partials == nil {
		tmpl, err = ParseString(data)
	} else {
		tmpl, err = ParseStringPartials(data, partials)
	}
	if err != nil {
		return "", err
	}
	return tmpl.Render(context...)
}

// RenderInLayout compiles a mustache template string and layout "wrapper" and
// uses the given data source - generally a map or struct - to render the
// compiled templates and return the output.
func RenderInLayout(data string, layoutData string, context ...interface{}) (string, error) {
	return RenderInLayoutPartials(data, layoutData, nil, context...)
}

func RenderInLayoutPartials(data string, layoutData string, partials PartialProvider, context ...interface{}) (string, error) {
	var layoutTmpl, tmpl *Template
	var err error
	if partials == nil {
		layoutTmpl, err = ParseString(layoutData)
	} else {
		layoutTmpl, err = ParseStringPartials(layoutData, partials)
	}
	if err != nil {
		return "", err
	}

	if partials == nil {
		tmpl, err = ParseString(data)
	} else {
		tmpl, err = ParseStringPartials(data, partials)
	}

	if err != nil {
		return "", err
	}

	return tmpl.RenderInLayout(layoutTmpl, context...)
}

// RenderFile loads a mustache template string from a file and compiles it, and
// then uses the the given data source - generally a map or struct - to render
// the template and return the output.
func RenderFile(filename string, context ...interface{}) (string, error) {
	tmpl, err := ParseFile(filename)
	if err != nil {
		return "", err
	}
	return tmpl.Render(context...)
}

// RenderFileInLayout loads a mustache template string and layout "wrapper"
// template string from files and compiles them, and  then uses the the given
// data source - generally a map or struct - to render the compiled templates
// and return the output.
func RenderFileInLayout(filename string, layoutFile string, context ...interface{}) (string, error) {
	layoutTmpl, err := ParseFile(layoutFile)
	if err != nil {
		return "", err
	}

	tmpl, err := ParseFile(filename)
	if err != nil {
		return "", err
	}
	return tmpl.RenderInLayout(layoutTmpl, context...)
}
