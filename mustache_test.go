package mustache

import (
    "container/vector"
    "os"
    "path"
    "strings"
    "testing"
)

type Test struct {
    tmpl     string
    context  interface{}
    expected string
}

type Data struct {
    a   bool
    b   string
}

type User struct {
    Name string
    Id   int64
}

type settings struct {
    Allow bool
}

func (u User) func1() string {
    return u.Name
}

func (u *User) func2() string {
    return u.Name
}

func (u *User) func3() (map[string]string, os.Error) {
    return map[string]string{"name": u.Name}, nil
}

func (u *User) func4() (map[string]string, os.Error) {
    return nil, nil
}

func (u *User) func5() (*settings, os.Error) {
    return &settings{true}, nil
}


func (u *User) func6() (*vector.Vector, os.Error) {
    var v vector.Vector
    v.Push(&settings{true})
    return &v, nil
}

func (u User) truefunc1() bool {
    return true
}

func (u *User) truefunc2() bool {
    return true
}

func makeVector(n int) *vector.Vector {
    v := new(vector.Vector)
    for i := 0; i < n; i++ {
        v.Push(&User{"Mike", 1})
    }
    return v
}

var tests = []Test{
    Test{`hello world`, nil, "hello world"},
    Test{`hello {{name}}`, map[string]string{"name": "world"}, "hello world"},
    Test{`{{a}}{{b}}{{c}}{{d}}`, map[string]string{"a": "a", "b": "b", "c": "c", "d": "d"}, "abcd"},
    Test{`0{{a}}1{{b}}23{{c}}456{{d}}89`, map[string]string{"a": "a", "b": "b", "c": "c", "d": "d"}, "0a1b23c456d89"},
    Test{`hello {{! comment }}world`, map[string]string{}, "hello world"},
    Test{`{{ a }}{{=<% %>=}}<%b %><%={{ }}=%>{{ c }}`, map[string]string{"a": "a", "b": "b", "c": "c"}, "abc"},
    Test{`{{ a }}{{= <% %> =}}<%b %><%= {{ }}=%>{{c}}`, map[string]string{"a": "a", "b": "b", "c": "c"}, "abc"},

    //does not exist
    Test{`{{dne}}`, map[string]string{"name": "world"}, ""},
    Test{`{{dne}}`, User{"Mike", 1}, ""},
    Test{`{{dne}}`, &User{"Mike", 1}, ""},
    Test{`{{#has}}{{/has}}`, &User{"Mike", 1}, ""},

    //section tests
    Test{`{{#a}}{{b}}{{/a}}`, Data{true, "hello"}, "hello"},
    Test{`{{#a}}{{b}}{{/a}}`, Data{false, "hello"}, ""},
    Test{`{{a}}{{#b}}{{b}}{{/b}}{{c}}`, map[string]string{"a": "a", "b": "b", "c": "c"}, "abc"},
    Test{`{{#a}}{{b}}{{/a}}`, struct {
        a []struct {
            b string
        }
    }{[]struct {
        b string
    }{struct{ b string }{"a"}, struct{ b string }{"b"}, struct{ b string }{"c"}}},
        "abc",
    },
    Test{`{{#a}}{{b}}{{/a}}`, struct{ a []map[string]string }{[]map[string]string{map[string]string{"b": "a"}, map[string]string{"b": "b"}, map[string]string{"b": "c"}}}, "abc"},

    Test{`{{#users}}{{Name}}{{/users}}`, map[string]interface{}{"users": []User{User{"Mike", 1}}}, "Mike"},

    Test{`{{#users}}gone{{Name}}{{/users}}`, map[string]interface{}{"users": nil}, ""},
    Test{`{{#users}}gone{{Name}}{{/users}}`, map[string]interface{}{"users": []User{}}, ""},
    Test{`{{#users}}{{Name}}{{/users}}`, map[string]interface{}{"users": []*User{&User{"Mike", 1}}}, "Mike"},
    Test{`{{#users}}{{Name}}{{/users}}`, map[string]interface{}{"users": vector.Vector([]interface{}{&User{"Mike", 12}})}, "Mike"},
    Test{`{{#users}}{{Name}}{{/users}}`, map[string]interface{}{"users": makeVector(1)}, "Mike"},
    Test{`{{Name}}`, User{"Mike", 1}, "Mike"},
    Test{`{{Name}}`, &User{"Mike", 1}, "Mike"},
    Test{"{{#users}}\n{{Name}}\n{{/users}}", map[string]interface{}{"users": makeVector(2)}, "Mike\nMike\n"},
    Test{"{{#users}}\r\n{{Name}}\r\n{{/users}}", map[string]interface{}{"users": makeVector(2)}, "Mike\r\nMike\r\n"},

    //inverted section tests
    Test{`{{a}}{{^b}}b{{/b}}{{c}}`, map[string]string{"a": "a", "c": "c"}, "abc"},
    Test{`{{a}}{{^b}}b{{/b}}{{c}}`, map[string]interface{}{"a": "a", "b": false, "c": "c"}, "abc"},
    Test{`{{^a}}b{{/a}}`, map[string]interface{}{"a": false}, "b"},
    Test{`{{^a}}b{{/a}}`, map[string]interface{}{"a": true}, ""},
    Test{`{{^a}}b{{/a}}`, map[string]interface{}{"a": "nonempty string"}, ""},

    //function tests
    Test{`{{#users}}{{func1}}{{/users}}`, map[string]interface{}{"users": []User{User{"Mike", 1}}}, "Mike"},
    Test{`{{#users}}{{func1}}{{/users}}`, map[string]interface{}{"users": []*User{&User{"Mike", 1}}}, "Mike"},
    Test{`{{#users}}{{func2}}{{/users}}`, map[string]interface{}{"users": []*User{&User{"Mike", 1}}}, "Mike"},

    Test{`{{#users}}{{#func3}}{{name}}{{/func3}}{{/users}}`, map[string]interface{}{"users": []*User{&User{"Mike", 1}}}, "Mike"},
    Test{`{{#users}}{{#func4}}{{name}}{{/func4}}{{/users}}`, map[string]interface{}{"users": []*User{&User{"Mike", 1}}}, ""},
    Test{`{{#truefunc1}}abcd{{/truefunc1}}`, User{"Mike", 1}, "abcd"},
    Test{`{{#truefunc1}}abcd{{/truefunc1}}`, &User{"Mike", 1}, "abcd"},
    Test{`{{#truefunc2}}abcd{{/truefunc2}}`, &User{"Mike", 1}, "abcd"},
    Test{`{{#func5}}{{#Allow}}abcd{{/Allow}}{{/func5}}`, &User{"Mike", 1}, "abcd"},
    Test{`{{#user}}{{#func5}}{{#Allow}}abcd{{/Allow}}{{/func5}}{{/user}}`, map[string]interface{}{"user": &User{"Mike", 1}}, "abcd"},
    Test{`{{#user}}{{#func6}}{{#Allow}}abcd{{/Allow}}{{/func6}}{{/user}}`, map[string]interface{}{"user": &User{"Mike", 1}}, "abcd"},

    //context chaining
    Test{`hello {{#section}}{{name}}{{/section}}`, map[string]interface{}{"section": map[string]string{"name": "world"}}, "hello world"},
    Test{`hello {{#section}}{{name}}{{/section}}`, map[string]interface{}{"name": "bob", "section": map[string]string{"name": "world"}}, "hello world"},
    Test{`hello {{#bool}}{{#section}}{{name}}{{/section}}{{/bool}}`, map[string]interface{}{"bool": true, "section": map[string]string{"name": "world"}}, "hello world"},
    Test{`{{#users}}{{canvas}}{{/users}}`, map[string]interface{}{"canvas": "hello", "users": []User{User{"Mike", 1}}}, "hello"},
}

func TestBasic(t *testing.T) {
    for _, test := range tests {
        output := Render(test.tmpl, test.context)
        if output != test.expected {
            t.Fatalf("%q expected %q got %q", test.tmpl, test.expected, output)
        }
    }
}

func TestFile(t *testing.T) {
    filename := path.Join(path.Join(os.Getenv("PWD"), "tests"), "test1.mustache")
    expected := "hello world"
    output := RenderFile(filename, map[string]string{"name": "world"})
    if output != expected {
        t.Fatalf("testfile expected %q got %q", expected, output)
    }
}

func TestPartial(t *testing.T) {
    filename := path.Join(path.Join(os.Getenv("PWD"), "tests"), "test2.mustache")
    expected := "hello world"
    output := RenderFile(filename, map[string]string{"Name": "world"})
    if output != expected {
        t.Fatalf("testpartial expected %q got %q", expected, output)
    }
}
func TestSectionPartial(t *testing.T) {
    filename := path.Join(path.Join(os.Getenv("PWD"), "tests"), "test3.mustache")
    expected := "Mike\nJoe\n"
    context := map[string]interface{}{"users": []User{User{"Mike", 1}, User{"Joe", 2}}}
    output := RenderFile(filename, context)
    if output != expected {
        t.Fatalf("testSectionPartial expected %q got %q", expected, output)
    }
}

func TestMultiContext(t *testing.T) {
    output := Render(`{{hello}} {{world}}`, map[string]string{"hello": "hello"}, struct{ world string }{"world"})
    output2 := Render(`{{hello}} {{world}}`, struct{ world string }{"world"}, map[string]string{"hello": "hello"})
    if output != "hello world" || output2 != "hello world" {
        t.Fatalf("TestMultiContext expected %q got %q", "hello world", output)
    }
}


var malformed = []Test{
    Test{`{{#a}}{{}}{{/a}}`, Data{true, "hello"}, "empty tag"},
    Test{`{{}}`, nil, "empty tag"},
    Test{`{{}`, nil, "unmatched open tag"},
    Test{`{{`, nil, "unmatched open tag"},
}

func TestMalformed(t *testing.T) {
    for _, test := range malformed {
        output := Render(test.tmpl, test.context)
        if strings.Index(output, test.expected) == -1 {
            t.Fatalf("%q expected %q in error %q", test.tmpl, test.expected, output)
        }
    }
}
