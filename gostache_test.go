package gostache

import (
    "testing"
)

type Test struct {
    tmpl     string
    context  interface{}
    expected string
}

var tests = []Test{
    Test{`hello {{name}}`, map[string]string{"name": "world"}, "hello world"},
    Test{`{{a}}{{b}}{{c}}{{d}}`, map[string]string{"a":"a","b":"b","c":"c","d":"d"}, "abcd"},
    Test{`0{{a}}1{{b}}23{{c}}456{{d}}89`, map[string]string{"a":"a","b":"b","c":"c","d":"d"}, "0a1b23c456d89"},
}


func TestBasic(t *testing.T) {

    for _, test := range (tests) {
        output := Render(test.tmpl, test.context)
        if output != test.expected {
            t.Fatalf("expected %q got %q", test.expected, output)
        }
    }
}
