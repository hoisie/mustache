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

type template struct {
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

func (tmpl *template) readString(s string) (string, os.Error) {
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

func (tmpl *template) parsePartial(name string) (*template, os.Error) {
    filename := path.Join(tmpl.dir, name+".mustache")

    partial, err := ParseFile(filename)

    if err != nil {
        return nil, err
    }

    return partial, nil
}

func (tmpl *template) parseSection(section *sectionElement) os.Error {
    for {
        text, err := tmpl.readString(tmpl.otag)

        if err == os.EOF {
            return parseError{section.startline, "Section " + section.name + " has no closing tag"}
        }

        // put text into an item
        text = text[0 : len(text)-len(tmpl.otag)]
        section.elems.Push(&textElement{strings.Bytes(text)})

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
                panicln("Invalid meta tag")
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

func (tmpl *template) parse() os.Error {
    for {
        text, err := tmpl.readString(tmpl.otag)

        if err == os.EOF {
            //put the remaining text in a block
            tmpl.elems.Push(&textElement{strings.Bytes(text)})
            return nil
        }

        // put text into an item
        text = text[0 : len(text)-len(tmpl.otag)]
        tmpl.elems.Push(&textElement{strings.Bytes(text)})

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
                panicln("Invalid meta tag")
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
        ret = val.FieldByName(name)
    }

    //if the lookup value is an interface, return the actual value
    if iface, ok := ret.(*reflect.InterfaceValue); ok && !iface.IsNil() {
        ret = iface.Elem()
    }

    return ret
}

func renderSection(section *sectionElement, context reflect.Value, buf io.Writer) {
    value := lookup(context, section.name)

    valueInd := reflect.Indirect(value)

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
    case *template:
        elem.renderTemplate(context, buf)
    }
}

func (tmpl *template) renderTemplate(context reflect.Value, buf io.Writer) {
    for i := 0; i < tmpl.elems.Len(); i++ {
        renderElement(tmpl.elems.At(i), context, buf)
    }
}

func (tmpl *template) Render(context interface{}, buf io.Writer) {
    val := reflect.NewValue(context)
    tmpl.renderTemplate(val, buf)
}

func ParseString(data string) (*template, os.Error) {
    cwd := os.Getenv("CWD")
    tmpl := template{data, "{{", "}}", 0, 1, cwd, new(vector.Vector)}
    err := tmpl.parse()

    if err != nil {
        return nil, err
    }

    return &tmpl, err
}

func ParseFile(filename string) (*template, os.Error) {
    data, err := ioutil.ReadFile(filename)

    if err != nil {
        return nil, err
    }

    dirname, _ := path.Split(filename)

    tmpl := template{string(data), "{{", "}}", 0, 1, dirname, new(vector.Vector)}
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
