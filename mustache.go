package mustache

import (
    "bytes"
    "container/vector"
    "fmt"
    "io"
    "io/ioutil"
    "os"
    "path"
    "reflect"
    "strings"
)

type textElement struct {
    text []byte
}

type varElement struct {
    name string
}

type sectionElement struct {
    name      string
    startline int
    elems     *vector.Vector
}

type Template struct {
    data    string
    otag    string
    ctag    string
    p       int
    curline int
    dir     string
    elems   *vector.Vector
}

type parseError struct {
    line    int
    message string
}

func (p parseError) String() string { return fmt.Sprintf("line %d: %s", p.line, p.message) }

func (tmpl *Template) readString(s string) (string, os.Error) {
    i := tmpl.p
    newlines := 0
    for true {
        //are we at the end of the string?
        if i+len(s) > len(tmpl.data) {
            return tmpl.data[tmpl.p:], os.EOF
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

func (tmpl *Template) parsePartial(name string) (*Template, os.Error) {
    filename := path.Join(tmpl.dir, name+".mustache")

    partial, err := ParseFile(filename)

    if err != nil {
        return nil, err
    }

    return partial, nil
}

func (tmpl *Template) parseSection(section *sectionElement) os.Error {
    for {
        text, err := tmpl.readString(tmpl.otag)

        if err == os.EOF {
            return parseError{section.startline, "Section " + section.name + " has no closing tag"}
        }

        // put text into an item
        text = text[0 : len(text)-len(tmpl.otag)]
        section.elems.Push(&textElement{[]byte(text)})

        text, err = tmpl.readString(tmpl.ctag)
        if err == os.EOF {
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
        case '#':
            name := strings.TrimSpace(tag[1:])

            //ignore the newline when a section starts
            if len(tmpl.data) > tmpl.p && tmpl.data[tmpl.p] == '\n' {
                tmpl.p += 1
            } else if len(tmpl.data) > tmpl.p+1 && tmpl.data[tmpl.p] == '\r' && tmpl.data[tmpl.p+1] == '\n' {
                tmpl.p += 2
            }

            se := sectionElement{name, tmpl.curline, new(vector.Vector)}
            err := tmpl.parseSection(&se)
            if err != nil {
                return err
            }
            section.elems.Push(&se)
        case '/':
            name := strings.TrimSpace(tag[1:])
            if name != section.name {
                return parseError{tmpl.curline, "interleaved closing tag: " + name}
            } else {
                return nil
            }
        case '>':
            name := strings.TrimSpace(tag[1:])
            partial, err := tmpl.parsePartial(name)
            if err != nil {
                return err
            }
            tmpl.elems.Push(partial)
        case '=':
            if tag[len(tag)-1] != '=' {
                return parseError{tmpl.curline, "Invalid meta tag"}
            }
            tag = strings.TrimSpace(tag[1 : len(tag)-1])
            newtags := strings.Split(tag, " ", 0)
            if len(newtags) == 2 {
                tmpl.otag = newtags[0]
                tmpl.ctag = newtags[1]
            }
        default:
            section.elems.Push(&varElement{tag})
        }
    }

    return nil
}

func (tmpl *Template) parse() os.Error {
    for {
        text, err := tmpl.readString(tmpl.otag)

        if err == os.EOF {
            //put the remaining text in a block
            tmpl.elems.Push(&textElement{[]byte(text)})
            return nil
        }

        // put text into an item
        text = text[0 : len(text)-len(tmpl.otag)]
        tmpl.elems.Push(&textElement{[]byte(text)})

        text, err = tmpl.readString(tmpl.ctag)
        if err == os.EOF {
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
        case '#':
            name := strings.TrimSpace(tag[1:])

            if len(tmpl.data) > tmpl.p && tmpl.data[tmpl.p] == '\n' {
                tmpl.p += 1
            } else if len(tmpl.data) > tmpl.p+1 && tmpl.data[tmpl.p] == '\r' && tmpl.data[tmpl.p+1] == '\n' {
                tmpl.p += 2
            }

            se := sectionElement{name, tmpl.curline, new(vector.Vector)}
            err := tmpl.parseSection(&se)
            if err != nil {
                return err
            }
            tmpl.elems.Push(&se)
        case '/':
            return parseError{tmpl.curline, "unmatched close tag"}
        case '>':
            name := strings.TrimSpace(tag[1:])
            partial, err := tmpl.parsePartial(name)
            if err != nil {
                return err
            }
            tmpl.elems.Push(partial)
        case '=':
            if tag[len(tag)-1] != '=' {
                return parseError{tmpl.curline, "Invalid meta tag"}
            }
            tag = strings.TrimSpace(tag[1 : len(tag)-1])
            newtags := strings.Split(tag, " ", 0)
            if len(newtags) == 2 {
                tmpl.otag = newtags[0]
                tmpl.ctag = newtags[1]
            }
        default:
            tmpl.elems.Push(&varElement{tag})
        }
    }

    return nil
}

// See if name is a method of the value at some level of indirection.
// The return values are the result of the call (which may be nil if
// there's trouble) and whether a method of the right name exists with
// any signature.
func callMethod(data reflect.Value, name string) (result reflect.Value, found bool) {
    found = false
    // Method set depends on pointerness, and the value may be arbitrarily
    // indirect.  Simplest approach is to walk down the pointer chain and
    // see if we can find the method at each step.
    // Most steps will see NumMethod() == 0.
    for {
        typ := data.Type()
        if nMethod := data.Type().NumMethod(); nMethod > 0 {
            for i := 0; i < nMethod; i++ {
                method := typ.Method(i)
                if method.Name == name {
                    found = true // we found the name regardless
                    // does receiver type match? (pointerness might be off)
                    if typ == method.Type.In(0) {
                        return call(data, method), found
                    }
                }
            }
        }
        if nd, ok := data.(*reflect.PtrValue); ok {
            data = nd.Elem()
        } else {
            break
        }
    }
    return
}

// Invoke the method. If its signature is wrong, return nil.
func call(v reflect.Value, method reflect.Method) reflect.Value {
    funcType := method.Type
    // Method must take no arguments, meaning as a func it has one argument (the receiver)
    if funcType.NumIn() != 1 {
        return nil
    }
    // Method must return a single value.
    if funcType.NumOut() != 1 {
        return nil
    }
    // Result will be the zeroth element of the returned slice.
    return method.Func.Call([]reflect.Value{v})[0]
}

func lookup(context reflect.Value, name string) reflect.Value {
    //if the context is an interface, get the actual value
    if iface, ok := context.(*reflect.InterfaceValue); ok && !iface.IsNil() {
        context = iface.Elem()
    }

    //the context may be a pointer, so do an indirect
    contextInd := reflect.Indirect(context)

    var ret reflect.Value = nil

    switch val := contextInd.(type) {
    case *reflect.MapValue:
        ret = val.Elem(reflect.NewValue(name))
    case *reflect.StructValue:
        //look for a field
        ret = val.FieldByName(name)
    }

    //look for a method
    if ret == nil {
        if result, found := callMethod(context, name); found {
            ret = result
        }
    }

    //if the lookup value is an interface, return the actual value
    if iface, ok := ret.(*reflect.InterfaceValue); ok && !iface.IsNil() {
        ret = iface.Elem()
    }

    return ret
}

func renderSection(section *sectionElement, context reflect.Value, buf io.Writer) {
    value := lookup(context, section.name)

    if value.Interface() == nil {
        return
    }

    valueInd := reflect.Indirect(value)
    //if the section is nil, we shouldn't do anything

    var contexts = new(vector.Vector)

    switch val := valueInd.(type) {
    case *reflect.BoolValue:
        if !val.Get() {
            return
        } else {
            contexts.Push(context)
        }
    case *reflect.SliceValue:
        for i := 0; i < val.Len(); i++ {
            contexts.Push(val.Elem(i))
        }
    case *reflect.ArrayValue:
        for i := 0; i < val.Len(); i++ {
            contexts.Push(val.Elem(i))
        }
    default:
        contexts.Push(context)
    }

    //by default we execute the section
    for j := 0; j < contexts.Len(); j++ {
        ctx := contexts.At(j).(reflect.Value)
        for i := 0; i < section.elems.Len(); i++ {
            renderElement(section.elems.At(i), ctx, buf)
        }
    }
}

func renderElement(element interface{}, context reflect.Value, buf io.Writer) {

    switch elem := element.(type) {
    case *textElement:
        buf.Write(elem.text)
    case *varElement:
        val := lookup(context, elem.name)
        if val != nil {
            fmt.Fprint(buf, val.Interface())
        }
    case *sectionElement:
        renderSection(elem, context, buf)
    case *Template:
        elem.renderTemplate(context, buf)
    }
}

func (tmpl *Template) renderTemplate(context reflect.Value, buf io.Writer) {
    for i := 0; i < tmpl.elems.Len(); i++ {
        renderElement(tmpl.elems.At(i), context, buf)
    }
}

func (tmpl *Template) Render(context interface{}, buf io.Writer) {
    val := reflect.NewValue(context)
    tmpl.renderTemplate(val, buf)
}

func ParseString(data string) (*Template, os.Error) {
    cwd := os.Getenv("CWD")
    tmpl := Template{data, "{{", "}}", 0, 1, cwd, new(vector.Vector)}
    err := tmpl.parse()

    if err != nil {
        return nil, err
    }

    return &tmpl, err
}

func ParseFile(filename string) (*Template, os.Error) {
    data, err := ioutil.ReadFile(filename)

    if err != nil {
        return nil, err
    }

    dirname, _ := path.Split(filename)

    tmpl := Template{string(data), "{{", "}}", 0, 1, dirname, new(vector.Vector)}
    err = tmpl.parse()

    if err != nil {
        return nil, err
    }

    return &tmpl, nil
}

func Render(data string, context interface{}) (string, os.Error) {
    tmpl, err := ParseString(data)

    if err != nil {
        return "", err
    }

    var buf bytes.Buffer
    tmpl.Render(context, &buf)

    return buf.String(), nil
}

func RenderFile(filename string, context interface{}) (string, os.Error) {
    tmpl, err := ParseFile(filename)

    if err != nil {
        return "", err
    }

    var buf bytes.Buffer
    tmpl.Render(context, &buf)

    return buf.String(), nil
}
