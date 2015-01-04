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
	"strings"
)

const defaultOtag = "{{"
const defaultCtag = "}}"

// environment is used to provide an environment and symbol table for recursive partials
type environment struct {
	partials map[string]*Template
}

func newEnvironment() environment {
	return environment{make(map[string]*Template)}
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
	rawBody   string
	otag      string
	ctag      string
}

type Template struct {
	data        string
	otag        string
	ctag        string
	p           int
	curline     int
	dir         string
	elems       []interface{}
	environment environment
}

type parseError struct {
	line    int
	message string
}

func (p parseError) Error() string { return fmt.Sprintf("line %d: %s", p.line, p.message) }

var (
	esc_quot = []byte("&quot;")
	esc_apos = []byte("&apos;")
	esc_amp  = []byte("&amp;")
	esc_lt   = []byte("&lt;")
	esc_gt   = []byte("&gt;")
)

func (s *sectionElement) writeRawBody(body string) {
	s.rawBody += body
}

func (tmpl *Template) readString(s string) (string, error) {
	i := tmpl.p
	newlines := 0
	for true {
		//are we at the end of the string?
		if i+len(s) > len(tmpl.data) {
			return tmpl.data[tmpl.p:], io.EOF
		}

		if tmpl.data[i] == '\n' {
			newlines++
		}

		if tmpl.data[i] != s[0] {
			i++
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
		} else {
			i++
		}
	}

	//should never be here
	return "", nil
}

func (tmpl *Template) lookupPartial(name string) *Template {
	result, ok := tmpl.environment.partials[name]
	if ok {
		return result
	}
	return nil
}

func (tmpl *Template) parsePartial(name string, indent string) (*Template, error) {
	filenames := []string{
		path.Join(tmpl.dir, name),
		path.Join(tmpl.dir, name+".mustache"),
		path.Join(tmpl.dir, name+".stache"),
		name,
		name + ".mustache",
		name + ".stache",
	}
	var filename string
	for _, name := range filenames {
		f, err := os.Open(name)
		if err == nil {
			filename = name
			f.Close()
			break
		}
	}
	if filename == "" {
		return ParseString("")
	}

	partial, err := parseFile(filename, indent, tmpl.environment)

	if err != nil {
		return nil, err
	}

	if len(indent) > 0 {
		partial.applyIndent(indent)
	}

	return partial, nil
}

func (tmpl *Template) applyIndent(indent string) {
	lastWasStandalone := false
	for i, element := range tmpl.elems {
		lastElement := i == len(tmpl.elems)-1
		switch elem := element.(type) {
		case *textElement:
			indentText(elem, indent, lastWasStandalone, lastElement)
			lastWasStandalone = false
		case *varElement:
			lastWasStandalone = true
		case *sectionElement:
		case *Template:
			elem.applyIndent(indent)
		}
	}
}

func indentText(elem *textElement, indent string, lastWasStandalone bool, lastElement bool) {
	var buf bytes.Buffer
	if !lastWasStandalone {
		buf.Write([]byte(indent))
	}
	oldBuf := elem.text
	n := len(oldBuf) - 1
	for i := 0; i < len(oldBuf); i++ {
		buf.Write([]byte{oldBuf[i]})
		if oldBuf[i] == '\n' && (i != n || !lastElement) {
			buf.Write([]byte(indent))
		}
	}

	elem.text = buf.Bytes()
}

func (tmpl *Template) parseSection(section *sectionElement) error {
	for {
		potentialStandalone := tmpl.isBeginningOfLine()
		text, err := tmpl.readString(tmpl.otag)

		if err == io.EOF {
			return parseError{section.startline, "Section " + section.name + " has no closing tag"}
		}

		if !potentialStandalone {
			potentialStandalone = strings.IndexByte(text, '\n') != -1
		}

		// put text into an item
		text = text[0 : len(text)-len(tmpl.otag)]

		// Store the text in case lambdas are used when rendering
		section.writeRawBody(text)

		section.elems = append(section.elems, &textElement{[]byte(text)})
		potentialStandalone = potentialStandalone && endsWithWhitespace([]byte(text))

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
			handleStandaloneLine(section.elems, tmpl, potentialStandalone)
			break
		case '#', '^':
			name := strings.TrimSpace(tag[1:])

			section.writeRawBody(tmpl.otag + tag + tmpl.ctag)

			handleStandaloneLine(section.elems, tmpl, potentialStandalone)

			se := sectionElement{name, tag[0] == '^', tmpl.curline, []interface{}{}, "", tmpl.otag, tmpl.ctag}
			err := tmpl.parseSection(&se)
			if err != nil {
				return err
			}
			section.elems = append(section.elems, &se)
		case '/':
			name := strings.TrimSpace(tag[1:])
			if name != section.name {
				return parseError{tmpl.curline, "interleaved closing tag: " + name}
			} else {
				handleStandaloneLine(section.elems, tmpl, potentialStandalone)
				return nil
			}
		case '>':
			name := strings.TrimSpace(tag[1:])

			section.writeRawBody(tmpl.otag + ">" + name + tmpl.ctag)

			indent := handleStandaloneLine(tmpl.elems, tmpl, potentialStandalone)
			partial := tmpl.lookupPartial(name)

			if partial == nil {
				partial, err = tmpl.parsePartial(name, indent)
				if err != nil {
					return err
				}
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
			handleStandaloneLine(section.elems, tmpl, potentialStandalone)
		case '{':
			// Remove the trailing '}' as well
			name := strings.TrimSpace(tag[1 : len(tag)-1])
			section.writeRawBody(tmpl.otag + "{" + name + "}" + tmpl.ctag)

			if tag[len(tag)-1] == '}' {
				//use a raw tag
				section.elems = append(section.elems, &varElement{name, true})
			}
		case '&':
			section.writeRawBody(tmpl.otag + tag + tmpl.ctag)
			section.elems = append(section.elems, &varElement{strings.TrimSpace(tag[1:]), true})
		default:
			section.writeRawBody(tmpl.otag + tag + tmpl.ctag)
			section.elems = append(section.elems, &varElement{tag, false})
		}
	}

	return nil
}

func (tmpl *Template) isBeginningOfLine() bool {
	return tmpl.p == 0 || tmpl.data[tmpl.p-1] == '\n'
}

func (tmpl *Template) parse() error {
	for {
		potentialStandalone := tmpl.isBeginningOfLine()
		text, err := tmpl.readString(tmpl.otag)
		if err == io.EOF {
			//put the remaining text in a block
			tmpl.elems = append(tmpl.elems, &textElement{[]byte(text)})
			return nil
		}

		if !potentialStandalone {
			potentialStandalone = strings.IndexByte(text, '\n') != -1
		}

		// put text into an item
		text = text[0 : len(text)-len(tmpl.otag)]
		tmpl.elems = append(tmpl.elems, &textElement{[]byte(text)})
		potentialStandalone = potentialStandalone && endsWithWhitespace([]byte(text))

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
			handleStandaloneLine(tmpl.elems, tmpl, potentialStandalone)
			//ignore comment
			break
		case '#', '^':
			name := strings.TrimSpace(tag[1:])
			handleStandaloneLine(tmpl.elems, tmpl, potentialStandalone)

			se := sectionElement{name, tag[0] == '^', tmpl.curline, []interface{}{}, "", tmpl.otag, tmpl.ctag}
			err := tmpl.parseSection(&se)
			if err != nil {
				return err
			}
			tmpl.elems = append(tmpl.elems, &se)
		case '/':
			return parseError{tmpl.curline, "unmatched close tag"}
		case '>':
			name := strings.TrimSpace(tag[1:])
			indent := handleStandaloneLine(tmpl.elems, tmpl, potentialStandalone)
			partial := tmpl.lookupPartial(name)

			if partial == nil {
				partial, err = tmpl.parsePartial(name, indent)
				if err != nil {
					return err
				}
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
			handleStandaloneLine(tmpl.elems, tmpl, potentialStandalone)
		case '{':
			//use a raw tag
			if tag[len(tag)-1] == '}' {
				tmpl.elems = append(tmpl.elems, &varElement{strings.TrimSpace(tag[1 : len(tag)-1]), true})
			}
		case '&':
			tmpl.elems = append(tmpl.elems, &varElement{strings.TrimSpace(tag[1:]), true})
		default:
			tmpl.elems = append(tmpl.elems, &varElement{tag, false})
		}
	}

	return nil
}

func handleStandaloneLine(elems []interface{}, tmpl *Template, potentialStandalone bool) string {
	if potentialStandalone {
		followingNewLine := tmpl.peekNewLine()
		if followingNewLine != -1 {
			tmpl.p += followingNewLine
			return removeIndentIfNecessary(elems)
		}
	}
	return ""
}

func removeIndentIfNecessary(elems []interface{}) string {
	if len(elems) > 0 {
		index := len(elems) - 1
		switch elems[index].(type) {
		case *textElement:
			{
				old := elems[index].(*textElement)
				newText, removed := removeTrailingIndent(string(old.text))
				elems[index] = &textElement{[]byte(newText)}
				return removed
			}
		}
	}
	return ""
}

func removeTrailingIndent(s string) (string, string) {
	removed := ""
	for i := len(s) - 1; i >= 0; {
		if s[i] == ' ' || s[i] == '\t' {
			removed = string(s[i]) + removed
			i--
			continue
		}
		return s[0 : i+1], removed
	}
	return "", s
}

func (tmpl *Template) isEndOfData() bool {
	return tmpl.p == len(tmpl.data)
}

func (tmpl *Template) peekNewLine() int {
	if tmpl.isEndOfData() {
		return 0
	}

	i := tmpl.p

	if tmpl.data[i] == '\n' {
		return 1
	}

	if tmpl.data[i] == '\r' {
		i++
		if tmpl.isEndOfData() {
			return 0
		}
		if tmpl.data[i] == '\n' {
			return 2
		}
	}

	return -1
}

func endsWithWhitespace(b []byte) bool {
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] == '\n' {
			return true
		}
		if b[i] == ' ' || b[i] == '\t' {
			continue
		}
		return false
	}
	return true
}

// Evaluate interfaces and pointers looking for a value that can look up the name, via a
// struct field, method, or map key, and return the result of the lookup.
func lookup(contextChain []interface{}, name string) reflect.Value {
	// dot notation
	if name != "." && strings.Contains(name, ".") {
		parts := strings.SplitN(name, ".", 2)

		v := lookup(contextChain, parts[0])
		return lookup([]interface{}{v}, parts[1])
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic while looking up %q: %s\n", name, r)
		}
	}()

Outer:
	for _, ctx := range contextChain { //i := len(contextChain) - 1; i >= 0; i-- {
		v := ctx.(reflect.Value)
		for v.IsValid() {
			typ := v.Type()
			if n := v.Type().NumMethod(); n > 0 {
				for i := 0; i < n; i++ {
					m := typ.Method(i)
					mtyp := m.Type
					if m.Name == name && mtyp.NumIn() == 1 {
						return v.Method(i).Call(nil)[0]
					}
				}
			}
			if name == "." {
				return v
			}
			switch av := v; av.Kind() {
			case reflect.Ptr:
				v = av.Elem()
			case reflect.Interface:
				v = av.Elem()
			case reflect.Struct:
				ret := av.FieldByName(name)
				if ret.IsValid() {
					return ret
				} else {
					continue Outer
				}
			case reflect.Map:
				ret := av.MapIndex(reflect.ValueOf(name))
				if ret.IsValid() {
					return ret
				} else {
					continue Outer
				}
			default:
				continue Outer
			}
		}
	}
	return reflect.Value{}
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

func renderSection(section *sectionElement, contextChain []interface{}, buf io.Writer) {
	value := lookup(contextChain, section.name)
	var context = contextChain[len(contextChain)-1].(reflect.Value)
	var contexts = []interface{}{}
	// if the value is nil, check if it's an inverted section
	isEmpty := isEmpty(value)
	if isEmpty && !section.inverted || !isEmpty && section.inverted {
		return
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
		case reflect.Func:
			out := val.Call([]reflect.Value{reflect.ValueOf(section.rawBody)})
			if len(out) > 0 && out[0].Kind() == reflect.String {
				content := evaluate(out[0].String(), section.otag, section.ctag, contextChain)

				section.elems = make([]interface{}, 0, 1)
				section.elems = append(section.elems, &textElement{[]byte(content)})
			}
			contexts = append(contexts, context)
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
}

func renderElement(element interface{}, contextChain []interface{}, buf io.Writer) {
	switch elem := element.(type) {
	case *textElement:
		buf.Write(elem.text)
	case *varElement:
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Panic while looking up %q: %s\n", elem.name, r)
			}
		}()
		val := lookup(contextChain, elem.name)

		if val.IsValid() {
			i := val.Interface()

			var content interface{}

			switch fn := reflect.ValueOf(i); fn.Kind() {
			case reflect.Func:
				out := fn.Call(nil)
				if len(out) > 0 && out[0].Kind() == reflect.String {
					content = evaluate(out[0].String(), defaultOtag, defaultCtag, contextChain)
				} else {
					content = ""
				}

			default:
				content = i
			}

			if elem.raw {
				fmt.Fprint(buf, content)
			} else {
				s := fmt.Sprint(content)
				template.HTMLEscape(buf, []byte(s))
			}
		}
	case *sectionElement:
		renderSection(elem, contextChain, buf)
	case *Template:
		elem.renderTemplate(contextChain, buf)
	}
}

func (tmpl *Template) renderTemplate(contextChain []interface{}, buf io.Writer) {
	for _, elem := range tmpl.elems {
		renderElement(elem, contextChain, buf)
	}
}

func (tmpl *Template) Render(context ...interface{}) string {
	var buf bytes.Buffer
	var contextChain []interface{}
	for _, c := range context {
		val := reflect.ValueOf(c)
		contextChain = append(contextChain, val)
	}
	tmpl.renderTemplate(contextChain, &buf)
	return buf.String()
}

func (tmpl *Template) RenderInLayout(layout *Template, context ...interface{}) string {
	content := tmpl.Render(context...)
	allContext := make([]interface{}, len(context)+1)
	copy(allContext[1:], context)
	allContext[0] = map[string]string{"content": content}
	return layout.Render(allContext...)
}

func ParseString(data string) (*Template, error) {
	return parseString(data, defaultOtag, defaultCtag, newEnvironment())
}

func parseString(data string, otag string, ctag string, environment environment) (*Template, error) {
	cwd := os.Getenv("CWD")
	tmpl := Template{data, otag, ctag, 0, 1, cwd, []interface{}{}, environment}
	err := tmpl.parse()

	if err != nil {
		return nil, err
	}

	return &tmpl, err
}

func ParseFile(filename string) (*Template, error) {
	return parseFile(filename, "", newEnvironment())
}

func parseFile(filename string, indent string, environment environment) (*Template, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	dirname, _ := path.Split(filename)

	basename := path.Base(filename)
	ext := path.Ext(filename)

	name := basename[0 : len(basename)-len(ext)]

	tmpl := Template{string(data), defaultOtag, defaultCtag, 0, 1, dirname, []interface{}{}, environment}
	tmpl.environment.partials[name] = &tmpl
	err = tmpl.parse()

	if err != nil {
		return nil, err
	}

	return &tmpl, nil
}

func evaluate(data string, otag string, ctag string, contextChain []interface{}) string {
	if tmpl, err := parseString(data, otag, ctag, newEnvironment()); err == nil {
		var buf bytes.Buffer
		tmpl.renderTemplate(contextChain, &buf)
		return buf.String()
	} else {
		return data
	}
}

func Render(data string, context ...interface{}) string {
	tmpl, err := ParseString(data)
	if err != nil {
		return err.Error()
	}
	return tmpl.Render(context...)
}

func RenderInLayout(data string, layoutData string, context ...interface{}) string {
	layoutTmpl, err := ParseString(layoutData)
	if err != nil {
		return err.Error()
	}
	tmpl, err := ParseString(data)
	if err != nil {
		return err.Error()
	}
	return tmpl.RenderInLayout(layoutTmpl, context...)
}

func RenderFile(filename string, context ...interface{}) string {
	tmpl, err := ParseFile(filename)
	if err != nil {
		return err.Error()
	}
	return tmpl.Render(context...)
}

func RenderFileInLayout(filename string, layoutFile string, context ...interface{}) string {
	layoutTmpl, err := ParseFile(layoutFile)
	if err != nil {
		return err.Error()
	}

	tmpl, err := ParseFile(filename)
	if err != nil {
		return err.Error()
	}
	return tmpl.RenderInLayout(layoutTmpl, context...)
}
