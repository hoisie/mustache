package gostache

import (
    "testing"
)

var basic = `hello {{name}}`

type Test struct {
    tmpl     string
    context  interface{}
    expected string
}

var tests = []Test{
    Test{basic, map[string]string{"name": "world"}, "hello world"},
}


func TestBasic(t *testing.T) {

    for _, test := range (tests) {
        output := Render(test.tmpl, test.context)
        if output != test.expected {
            t.Fatalf("expected %q got %q", test.expected, output)
        }
    }
}
