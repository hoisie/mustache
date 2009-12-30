package gostache

import (
    "bytes"
    "container/vector"
    "io"
    "os"
    "reflect"
    "strings"
)

type textElement struct {
    text []byte
}

type varElement struct {
    name string
}

type template struct {
    data  string
    otag  string
    ctag  string
    p     int
    elems *vector.Vector
}

func (tmpl *template) readString(s string) (string, os.Error) {
    i := tmpl.p
    for true {
        //are we at the end of the string?
        if i+len(s) > len(tmpl.data) {
            return "", os.EOF
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
            return text, nil
        } else {
            i++
        }
    }

    //should never be here
    return "", nil
}

func (tmpl *template) readToEnd() string {
    text := tmpl.data[tmpl.p:]
    tmpl.p = len(tmpl.data)
    return text
}


func (tmpl *template) parse() {
    for {
        text, err := tmpl.readString(tmpl.otag)

        if err == os.EOF {
            //put the remaining text in a block
            remaining := tmpl.readToEnd()
            tmpl.elems.Push(&textElement{strings.Bytes(remaining)})
            return
        }

        // put text into an item
        text = text[0 : len(text)-len(tmpl.otag)]
        tmpl.elems.Push(&textElement{strings.Bytes(text)})

        text, err = tmpl.readString(tmpl.ctag)
        if err == os.EOF {
            //put the remaining text in a block
            panicln("unmatched open tag")
        }

        //trim the close tag off the text
        name := strings.TrimSpace(text[0 : len(text)-len(tmpl.ctag)])

        tmpl.elems.Push(&varElement{name})
    }
}

func lookup(context reflect.Value, name string) reflect.Value {
    switch val := context.(type) {
    case *reflect.MapValue:
        return val.Elem(reflect.NewValue(name))
    }

    return nil
}

func (tmpl *template) execute(context reflect.Value, buf io.Writer) {

    for i := 0; i < tmpl.elems.Len(); i++ {
        switch elem := tmpl.elems.At(i).(type) {
        case *textElement:
            buf.Write(elem.text)
        case *varElement:
            val := lookup(context, elem.name)
            if val != nil {
                buf.Write(strings.Bytes(val.(*reflect.StringValue).Get()))
            }
        }
    }
}

func Render(data string, context interface{}) string {
    parser := template{data, "{{", "}}", 0, new(vector.Vector)}
    parser.parse()
    val := reflect.NewValue(context)
    var buf bytes.Buffer
    parser.execute(val, &buf)
    return buf.String()
}
