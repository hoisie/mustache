package mustache

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"strings"
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
	ID   int64
}

type Settings struct {
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

func (u *User) Func5() (*Settings, error) {
	return &Settings{true}, nil
}

func (u *User) Func6() ([]interface{}, error) {
	var v []interface{}
	v = append(v, &Settings{true})
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
	{`{{var}}`, map[string]string{"var": "& \" < >"}, "&amp; &#34; &lt; &gt;", nil},
	{`{{{var}}}`, map[string]string{"var": "& \" < >"}, "& \" < >", nil},
	{`{{a}}{{b}}{{c}}{{d}}`, map[string]string{"a": "a", "b": "b", "c": "c", "d": "d"}, "abcd", nil},
	{`0{{a}}1{{b}}23{{c}}456{{d}}89`, map[string]string{"a": "a", "b": "b", "c": "c", "d": "d"}, "0a1b23c456d89", nil},
	{`hello {{! comment }}world`, map[string]string{}, "hello world", nil},
	{`{{ a }}{{=<% %>=}}<%b %><%={{ }}=%>{{ c }}`, map[string]string{"a": "a", "b": "b", "c": "c"}, "abc", nil},
	{`{{ a }}{{= <% %> =}}<%b %><%= {{ }}=%>{{c}}`, map[string]string{"a": "a", "b": "b", "c": "c"}, "abc", nil},

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
	{"{{#users}}Hi {{Name}}{{/users}}", map[string]interface{}{"users": ""}, "", nil},
	{"{{#users}}Hi {{Name}}{{/users}}", map[string]interface{}{"users": []interface{}{}}, "", nil},
	{"{{#users}}Hi {{Name}}{{/users}}", map[string]interface{}{"users": false}, "", nil},

	//section does not exist
	{`{{#has}}{{/has}}`, &User{"Mike", 1}, "", nil},

	// implicit iterator tests
	{`"{{#list}}({{.}}){{/list}}"`, map[string]interface{}{"list": []string{"a", "b", "c", "d", "e"}}, "\"(a)(b)(c)(d)(e)\"", nil},
	{`"{{#list}}({{.}}){{/list}}"`, map[string]interface{}{"list": []int{1, 2, 3, 4, 5}}, "\"(1)(2)(3)(4)(5)\"", nil},
	{`"{{#list}}({{.}}){{/list}}"`, map[string]interface{}{"list": []float64{1.10, 2.20, 3.30, 4.40, 5.50}}, "\"(1.1)(2.2)(3.3)(4.4)(5.5)\"", nil},

	//inverted section tests
	{`{{a}}{{^b}}b{{/b}}{{c}}`, map[string]interface{}{"a": "a", "b": false, "c": "c"}, "abc", nil},
	{`{{^a}}b{{/a}}`, map[string]interface{}{"a": false}, "b", nil},
	{`{{^a}}b{{/a}}`, map[string]interface{}{"a": true}, "", nil},
	{`{{^a}}b{{/a}}`, map[string]interface{}{"a": "nonempty string"}, "", nil},
	{`{{^a}}b{{/a}}`, map[string]interface{}{"a": []string{}}, "b", nil},
	{`{{a}}{{^b}}b{{/b}}{{c}}`, map[string]string{"a": "a", "c": "c"}, "abc", nil},

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
	{`"{{#a}}{{b.c.d.e.name}}{{/a}}" == "Phil"`, map[string]interface{}{"a": map[string]interface{}{"b": map[string]interface{}{"c": map[string]interface{}{"d": map[string]interface{}{"e": map[string]string{"name": "Phil"}}}}}, "b": map[string]interface{}{"c": map[string]interface{}{"d": map[string]interface{}{"e": map[string]string{"name": "Wrong"}}}}}, `"Phil" == "Phil"`, nil},
	{`{{#a}}{{b.c}}{{/a}}`, map[string]interface{}{"a": map[string]interface{}{"b": map[string]string{}}, "b": map[string]string{"c": "ERROR"}}, "", nil},
}

func TestBasic(t *testing.T) {
	// Default behavior, AllowMissingVariables=true
	for _, test := range tests {
		output, err := Render(test.tmpl, test.context)
		if err != nil {
			t.Errorf("%q expected %q but got error %q", test.tmpl, test.expected, err.Error())
		} else if output != test.expected {
			t.Errorf("%q expected %q got %q", test.tmpl, test.expected, output)
		}
	}

	// Now set AllowMissingVariables=false and test again
	AllowMissingVariables = false
	defer func() { AllowMissingVariables = true }()
	for _, test := range tests {
		output, err := Render(test.tmpl, test.context)
		if err != nil {
			t.Errorf("%s expected %s but got error %s", test.tmpl, test.expected, err.Error())
		} else if output != test.expected {
			t.Errorf("%q expected %q got %q", test.tmpl, test.expected, output)
		}
	}
}

var missing = []Test{
	//does not exist
	{`{{dne}}`, map[string]string{"name": "world"}, "", nil},
	{`{{dne}}`, User{"Mike", 1}, "", nil},
	{`{{dne}}`, &User{"Mike", 1}, "", nil},
	//dotted names(dot notation)
	{`"{{a.b.c}}" == ""`, map[string]interface{}{}, `"" == ""`, nil},
	{`"{{a.b.c.name}}" == ""`, map[string]interface{}{"a": map[string]interface{}{"b": map[string]string{}}, "c": map[string]string{"name": "Jim"}}, `"" == ""`, nil},
}

func TestMissing(t *testing.T) {
	// Default behavior, AllowMissingVariables=true
	for _, test := range missing {
		output, err := Render(test.tmpl, test.context)
		if err != nil {
			t.Error(err)
		} else if output != test.expected {
			t.Errorf("%q expected %q got %q", test.tmpl, test.expected, output)
		}
	}

	// Now set AllowMissingVariables=false and confirm we get errors.
	AllowMissingVariables = false
	defer func() { AllowMissingVariables = true }()
	for _, test := range missing {
		output, err := Render(test.tmpl, test.context)
		if err == nil {
			t.Errorf("%q expected missing variable error but got %q", test.tmpl, output)
		} else if strings.Index(err.Error(), "Missing variable") == -1 {
			t.Errorf("%q expected missing variable error but got %q", test.tmpl, err.Error())
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

func TestFRender(t *testing.T) {
	filename := path.Join(path.Join(os.Getenv("PWD"), "tests"), "test1.mustache")
	expected := "hello world"
	tmpl, err := ParseFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	err = tmpl.FRender(&buf, map[string]string{"name": "world"})
	if err != nil {
		t.Fatal(err)
	}
	output := buf.String()
	if output != expected {
		t.Fatalf("testfile expected %q got %q", expected, output)
	}
}

func TestPartial(t *testing.T) {
	filename := path.Join(path.Join(os.Getenv("PWD"), "tests"), "test2.mustache")
	expected := "hello world"
	tmpl, err := ParseFile(filename)
	if err != nil {
		t.Error(err)
		return
	}
	output, err := tmpl.Render(map[string]string{"Name": "world"})
	if err != nil {
		t.Error(err)
		return
	} else if output != expected {
		t.Errorf("testpartial expected %q got %q", expected, output)
		return
	}

	expectedTags := []tag{
		{
			Type: Partial,
			Name: "partial",
		},
	}
	compareTags(t, tmpl.Tags(), expectedTags)
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

func TestLayoutToWriter(t *testing.T) {
	for _, test := range layoutTests {
		tmpl, err := ParseString(test.tmpl)
		if err != nil {
			t.Error(err)
			continue
		}
		layoutTmpl, err := ParseString(test.layout)
		if err != nil {
			t.Error(err)
			continue
		}
		var buf bytes.Buffer
		err = tmpl.FRenderInLayout(&buf, layoutTmpl, test.context)
		if err != nil {
			t.Error(err)
		} else if buf.String() != test.expected {
			t.Errorf("%q expected %q got %q", test.tmpl, test.expected, buf.String())
		}
	}
}

type Person struct {
	FirstName string
	LastName  string
}

func (p *Person) Name1() string {
	return p.FirstName + " " + p.LastName
}

func (p Person) Name2() string {
	return p.FirstName + " " + p.LastName
}

func TestPointerReceiver(t *testing.T) {
	p := Person{"John", "Smith"}
	tests := []struct {
		tmpl     string
		context  interface{}
		expected string
	}{
		{
			tmpl:     "{{Name1}}",
			context:  &p,
			expected: "John Smith",
		},
		{
			tmpl:     "{{Name2}}",
			context:  &p,
			expected: "John Smith",
		},
		{
			tmpl:     "{{Name1}}",
			context:  p,
			expected: "",
		},
		{
			tmpl:     "{{Name2}}",
			context:  p,
			expected: "John Smith",
		},
	}
	for _, test := range tests {
		output, err := Render(test.tmpl, test.context)
		if err != nil {
			t.Error(err)
		} else if output != test.expected {
			t.Errorf("expected %q got %q", test.expected, output)
		}
	}
}

type tag struct {
	Type TagType
	Name string
	Tags []tag
}

type tagsTest struct {
	tmpl string
	tags []tag
}

var tagTests = []tagsTest{
	{
		tmpl: `hello world`,
		tags: nil,
	},
	{
		tmpl: `hello {{name}}`,
		tags: []tag{
			{
				Type: Variable,
				Name: "name",
			},
		},
	},
	{
		tmpl: `{{#name}}hello {{name}}{{/name}}{{^name}}hello {{name2}}{{/name}}`,
		tags: []tag{
			{
				Type: Section,
				Name: "name",
				Tags: []tag{
					{
						Type: Variable,
						Name: "name",
					},
				},
			},
			{
				Type: InvertedSection,
				Name: "name",
				Tags: []tag{
					{
						Type: Variable,
						Name: "name2",
					},
				},
			},
		},
	},
}

func TestTags(t *testing.T) {
	for _, test := range tagTests {
		testTags(t, &test)
	}
}

func testTags(t *testing.T, test *tagsTest) {
	tmpl, err := ParseString(test.tmpl)
	if err != nil {
		t.Error(err)
		return
	}
	compareTags(t, tmpl.Tags(), test.tags)
}

func compareTags(t *testing.T, actual []Tag, expected []tag) {
	if len(actual) != len(expected) {
		t.Errorf("expected %d tags, got %d", len(expected), len(actual))
		return
	}
	for i, tag := range actual {
		if tag.Type() != expected[i].Type {
			t.Errorf("expected %s, got %s", expected[i].Type, tag.Type())
			return
		}
		if tag.Name() != expected[i].Name {
			t.Errorf("expected %s, got %s", expected[i].Name, tag.Name())
			return
		}

		switch tag.Type() {
		case Variable:
			if len(expected[i].Tags) != 0 {
				t.Errorf("expected %d tags, got 0", len(expected[i].Tags))
				return
			}
		case Section, InvertedSection:
			compareTags(t, tag.Tags(), expected[i].Tags)
		case Partial:
			compareTags(t, tag.Tags(), expected[i].Tags)
		case Invalid:
			t.Errorf("invalid tag type: %s", tag.Type())
			return
		default:
			t.Errorf("invalid tag type: %s", tag.Type())
			return
		}
	}
}
