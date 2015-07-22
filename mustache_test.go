package mustache

import (
	"fmt"
	"os"
	"path"
	"testing"
)

type Test struct {
	tmpl     string
	context  interface{}
	expected string
	err      error
}

type Data struct {
	A bool
	B string
}

type User struct {
	Name string
	Id   int64
}

type settings struct {
	Allow bool
}

func (u User) Func1() string {
	return u.Name
}

func (u *User) Func2() string {
	return u.Name
}

func (u *User) Func3() (map[string]string, error) {
	return map[string]string{"name": u.Name}, nil
}

func (u *User) Func4() (map[string]string, error) {
	return nil, nil
}

func (u *User) Func5() (*settings, error) {
	return &settings{true}, nil
}

func (u *User) Func6() ([]interface{}, error) {
	var v []interface{}
	v = append(v, &settings{true})
	return v, nil
}

func (u User) Truefunc1() bool {
	return true
}

func (u *User) Truefunc2() bool {
	return true
}

func makeVector(n int) []interface{} {
	var v []interface{}
	for i := 0; i < n; i++ {
		v = append(v, &User{"Mike", 1})
	}
	return v
}

type Category struct {
	Tag         string
	Description string
}

func (c Category) DisplayName() string {
	return c.Tag + " - " + c.Description
}

var tests = []Test{
	{`hello world`, nil, "hello world", nil},
	{`hello {{name}}`, map[string]string{"name": "world"}, "hello world", nil},
	{`{{var}}`, map[string]string{"var": "5 > 2"}, "5 &gt; 2", nil},
	{`{{{var}}}`, map[string]string{"var": "5 > 2"}, "5 > 2", nil},
	{`{{a}}{{b}}{{c}}{{d}}`, map[string]string{"a": "a", "b": "b", "c": "c", "d": "d"}, "abcd", nil},
	{`0{{a}}1{{b}}23{{c}}456{{d}}89`, map[string]string{"a": "a", "b": "b", "c": "c", "d": "d"}, "0a1b23c456d89", nil},
	{`hello {{! comment }}world`, map[string]string{}, "hello world", nil},
	{`{{ a }}{{=<% %>=}}<%b %><%={{ }}=%>{{ c }}`, map[string]string{"a": "a", "b": "b", "c": "c"}, "abc", nil},
	{`{{ a }}{{= <% %> =}}<%b %><%= {{ }}=%>{{c}}`, map[string]string{"a": "a", "b": "b", "c": "c"}, "abc", nil},

	//does not exist
	{`{{dne}}`, map[string]string{"name": "world"}, "", nil},
	{`{{dne}}`, User{"Mike", 1}, "", nil},
	{`{{dne}}`, &User{"Mike", 1}, "", nil},
	{`{{#has}}{{/has}}`, &User{"Mike", 1}, "", nil},

	//section tests
	{`{{#A}}{{B}}{{/A}}`, Data{true, "hello"}, "hello", nil},
	{`{{#A}}{{{B}}}{{/A}}`, Data{true, "5 > 2"}, "5 > 2", nil},
	{`{{#A}}{{B}}{{/A}}`, Data{true, "5 > 2"}, "5 &gt; 2", nil},
	{`{{#A}}{{B}}{{/A}}`, Data{false, "hello"}, "", nil},
	{`{{a}}{{#b}}{{b}}{{/b}}{{c}}`, map[string]string{"a": "a", "b": "b", "c": "c"}, "abc", nil},
	{`{{#A}}{{B}}{{/A}}`, struct {
		A []struct {
			B string
		}
	}{[]struct {
		B string
	}{{"a"}, {"b"}, {"c"}}},
		"abc",
		nil,
	},
	{`{{#A}}{{b}}{{/A}}`, struct{ A []map[string]string }{[]map[string]string{{"b": "a"}, {"b": "b"}, {"b": "c"}}}, "abc", nil},

	{`{{#users}}{{Name}}{{/users}}`, map[string]interface{}{"users": []User{{"Mike", 1}}}, "Mike", nil},

	{`{{#users}}gone{{Name}}{{/users}}`, map[string]interface{}{"users": nil}, "", nil},
	{`{{#users}}gone{{Name}}{{/users}}`, map[string]interface{}{"users": (*User)(nil)}, "", nil},
	{`{{#users}}gone{{Name}}{{/users}}`, map[string]interface{}{"users": []User{}}, "", nil},

	{`{{#users}}{{Name}}{{/users}}`, map[string]interface{}{"users": []*User{{"Mike", 1}}}, "Mike", nil},
	{`{{#users}}{{Name}}{{/users}}`, map[string]interface{}{"users": []interface{}{&User{"Mike", 12}}}, "Mike", nil},
	{`{{#users}}{{Name}}{{/users}}`, map[string]interface{}{"users": makeVector(1)}, "Mike", nil},
	{`{{Name}}`, User{"Mike", 1}, "Mike", nil},
	{`{{Name}}`, &User{"Mike", 1}, "Mike", nil},
	{"{{#users}}\n{{Name}}\n{{/users}}", map[string]interface{}{"users": makeVector(2)}, "Mike\nMike\n", nil},
	{"{{#users}}\r\n{{Name}}\r\n{{/users}}", map[string]interface{}{"users": makeVector(2)}, "Mike\r\nMike\r\n", nil},

	// implicit iterator tests
	{`"{{#list}}({{.}}){{/list}}"`, map[string]interface{}{"list": []string{"a", "b", "c", "d", "e"}}, "\"(a)(b)(c)(d)(e)\"", nil},
	{`"{{#list}}({{.}}){{/list}}"`, map[string]interface{}{"list": []int{1, 2, 3, 4, 5}}, "\"(1)(2)(3)(4)(5)\"", nil},
	{`"{{#list}}({{.}}){{/list}}"`, map[string]interface{}{"list": []float64{1.10, 2.20, 3.30, 4.40, 5.50}}, "\"(1.1)(2.2)(3.3)(4.4)(5.5)\"", nil},

	//inverted section tests
	{`{{a}}{{^b}}b{{/b}}{{c}}`, map[string]string{"a": "a", "c": "c"}, "abc", nil},
	{`{{a}}{{^b}}b{{/b}}{{c}}`, map[string]interface{}{"a": "a", "b": false, "c": "c"}, "abc", nil},
	{`{{^a}}b{{/a}}`, map[string]interface{}{"a": false}, "b", nil},
	{`{{^a}}b{{/a}}`, map[string]interface{}{"a": true}, "", nil},
	{`{{^a}}b{{/a}}`, map[string]interface{}{"a": "nonempty string"}, "", nil},
	{`{{^a}}b{{/a}}`, map[string]interface{}{"a": []string{}}, "b", nil},

	//function tests
	{`{{#users}}{{Func1}}{{/users}}`, map[string]interface{}{"users": []User{{"Mike", 1}}}, "Mike", nil},
	{`{{#users}}{{Func1}}{{/users}}`, map[string]interface{}{"users": []*User{{"Mike", 1}}}, "Mike", nil},
	{`{{#users}}{{Func2}}{{/users}}`, map[string]interface{}{"users": []*User{{"Mike", 1}}}, "Mike", nil},

	{`{{#users}}{{#Func3}}{{name}}{{/Func3}}{{/users}}`, map[string]interface{}{"users": []*User{{"Mike", 1}}}, "Mike", nil},
	{`{{#users}}{{#Func4}}{{name}}{{/Func4}}{{/users}}`, map[string]interface{}{"users": []*User{{"Mike", 1}}}, "", nil},
	{`{{#Truefunc1}}abcd{{/Truefunc1}}`, User{"Mike", 1}, "abcd", nil},
	{`{{#Truefunc1}}abcd{{/Truefunc1}}`, &User{"Mike", 1}, "abcd", nil},
	{`{{#Truefunc2}}abcd{{/Truefunc2}}`, &User{"Mike", 1}, "abcd", nil},
	{`{{#Func5}}{{#Allow}}abcd{{/Allow}}{{/Func5}}`, &User{"Mike", 1}, "abcd", nil},
	{`{{#user}}{{#Func5}}{{#Allow}}abcd{{/Allow}}{{/Func5}}{{/user}}`, map[string]interface{}{"user": &User{"Mike", 1}}, "abcd", nil},
	{`{{#user}}{{#Func6}}{{#Allow}}abcd{{/Allow}}{{/Func6}}{{/user}}`, map[string]interface{}{"user": &User{"Mike", 1}}, "abcd", nil},

	//context chaining
	{`hello {{#section}}{{name}}{{/section}}`, map[string]interface{}{"section": map[string]string{"name": "world"}}, "hello world", nil},
	{`hello {{#section}}{{name}}{{/section}}`, map[string]interface{}{"name": "bob", "section": map[string]string{"name": "world"}}, "hello world", nil},
	{`hello {{#bool}}{{#section}}{{name}}{{/section}}{{/bool}}`, map[string]interface{}{"bool": true, "section": map[string]string{"name": "world"}}, "hello world", nil},
	{`{{#users}}{{canvas}}{{/users}}`, map[string]interface{}{"canvas": "hello", "users": []User{{"Mike", 1}}}, "hello", nil},
	{`{{#categories}}{{DisplayName}}{{/categories}}`, map[string][]*Category{
		"categories": {&Category{"a", "b"}},
	}, "a - b", nil},

	//dotted names(dot notation)
	{`"{{person.name}}" == "{{#person}}{{name}}{{/person}}"`, map[string]interface{}{"person": map[string]string{"name": "Joe"}}, `"Joe" == "Joe"`, nil},
	{`"{{{person.name}}}" == "{{#person}}{{{name}}}{{/person}}"`, map[string]interface{}{"person": map[string]string{"name": "Joe"}}, `"Joe" == "Joe"`, nil},
	{`"{{a.b.c.d.e.name}}" == "Phil"`, map[string]interface{}{"a": map[string]interface{}{"b": map[string]interface{}{"c": map[string]interface{}{"d": map[string]interface{}{"e": map[string]string{"name": "Phil"}}}}}}, `"Phil" == "Phil"`, nil},
	{`"{{a.b.c}}" == ""`, map[string]interface{}{}, `"" == ""`, nil},
	{`"{{a.b.c.name}}" == ""`, map[string]interface{}{"a": map[string]interface{}{"b": map[string]string{}}, "c": map[string]string{"name": "Jim"}}, `"" == ""`, nil},
	{`"{{#a}}{{b.c.d.e.name}}{{/a}}" == "Phil"`, map[string]interface{}{"a": map[string]interface{}{"b": map[string]interface{}{"c": map[string]interface{}{"d": map[string]interface{}{"e": map[string]string{"name": "Phil"}}}}}, "b": map[string]interface{}{"c": map[string]interface{}{"d": map[string]interface{}{"e": map[string]string{"name": "Wrong"}}}}}, `"Phil" == "Phil"`, nil},
	{`{{#a}}{{b.c}}{{/a}}`, map[string]interface{}{"a": map[string]interface{}{"b": map[string]string{}}, "b": map[string]string{"c": "ERROR"}}, "", nil},
}

func TestBasic(t *testing.T) {
	for _, test := range tests {
		output, err := Render(test.tmpl, test.context)
		if err != nil {
			t.Error(err)
		} else if output != test.expected {
			t.Errorf("%q expected %q got %q", test.tmpl, test.expected, output)
		}
	}
}

func TestFile(t *testing.T) {
	filename := path.Join(path.Join(os.Getenv("PWD"), "tests"), "test1.mustache")
	expected := "hello world"
	output, err := RenderFile(filename, map[string]string{"name": "world"})
	if err != nil {
		t.Error(err)
	} else if output != expected {
		t.Errorf("testfile expected %q got %q", expected, output)
	}
}

func TestPartial(t *testing.T) {
	filename := path.Join(path.Join(os.Getenv("PWD"), "tests"), "test2.mustache")
	println(filename)
	expected := "hello world"
	output, err := RenderFile(filename, map[string]string{"Name": "world"})
	if err != nil {
		t.Error(err)
	} else if output != expected {
		t.Errorf("testpartial expected %q got %q", expected, output)
	}
}

/*
func TestSectionPartial(t *testing.T) {
    filename := path.Join(path.Join(os.Getenv("PWD"), "tests"), "test3.mustache")
    expected := "Mike\nJoe\n"
    context := map[string]interface{}{"users": []User{{"Mike", 1}, {"Joe", 2}}}
    output := RenderFile(filename, context)
    if output != expected {
        t.Fatalf("testSectionPartial expected %q got %q", expected, output)
    }
}
*/
func TestMultiContext(t *testing.T) {
	output, err := Render(`{{hello}} {{World}}`, map[string]string{"hello": "hello"}, struct{ World string }{"world"})
	if err != nil {
		t.Error(err)
		return
	}
	output2, err := Render(`{{hello}} {{World}}`, struct{ World string }{"world"}, map[string]string{"hello": "hello"})
	if err != nil {
		t.Error(err)
		return
	}
	if output != "hello world" || output2 != "hello world" {
		t.Errorf("TestMultiContext expected %q got %q", "hello world", output)
		return
	}
}

var malformed = []Test{
	{`{{#a}}{{}}{{/a}}`, Data{true, "hello"}, "", fmt.Errorf("line 1: empty tag")},
	{`{{}}`, nil, "", fmt.Errorf("line 1: empty tag")},
	{`{{}`, nil, "", fmt.Errorf("line 1: unmatched open tag")},
	{`{{`, nil, "", fmt.Errorf("line 1: unmatched open tag")},
	//invalid syntax - https://github.com/hoisie/mustache/issues/10
	{`{{#a}}{{#b}}{{/a}}{{/b}}}`, map[string]interface{}{}, "", fmt.Errorf("line 1: interleaved closing tag: a")},
}

func TestMalformed(t *testing.T) {
	for _, test := range malformed {
		output, err := Render(test.tmpl, test.context)
		if err != nil {
			if test.err == nil {
				t.Error(err)
			} else if test.err.Error() != err.Error() {
				t.Errorf("%q expected error %q but got error %q", test.tmpl, test.err.Error(), err.Error())
			}
		} else {
			if test.err == nil {
				t.Errorf("%q expected %q got %q", test.tmpl, test.expected, output)
			} else {
				t.Errorf("%q expected error %q but got %q", test.tmpl, test.err.Error(), output)
			}
		}
	}
}

type LayoutTest struct {
	layout   string
	tmpl     string
	context  interface{}
	expected string
}

var layoutTests = []LayoutTest{
	{`Header {{content}} Footer`, `Hello World`, nil, `Header Hello World Footer`},
	{`Header {{content}} Footer`, `Hello {{s}}`, map[string]string{"s": "World"}, `Header Hello World Footer`},
	{`Header {{content}} Footer`, `Hello {{content}}`, map[string]string{"content": "World"}, `Header Hello World Footer`},
	{`Header {{extra}} {{content}} Footer`, `Hello {{content}}`, map[string]string{"content": "World", "extra": "extra"}, `Header extra Hello World Footer`},
	{`Header {{content}} {{content}} Footer`, `Hello {{content}}`, map[string]string{"content": "World"}, `Header Hello World Hello World Footer`},
}

func TestLayout(t *testing.T) {
	for _, test := range layoutTests {
		output, err := RenderInLayout(test.tmpl, test.layout, test.context)
		if err != nil {
			t.Error(err)
		} else if output != test.expected {
			t.Errorf("%q expected %q got %q", test.tmpl, test.expected, output)
		}
	}
}
