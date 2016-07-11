# Mustache Template Engine for Go

[![Build Status](https://img.shields.io/travis/cbroglie/mustache.svg)](https://travis-ci.org/cbroglie/mustache)

## Why a Fork?

I forked [hoisie/mustache](https://github.com/hoisie/mustache) because it does not appear to be maintained, and I wanted to add the following functionality:
- Update the API to follow the idiomatic Go convention of returning errors (this is a breaking change)
- Add option to treat missing variables as errors

## Overview

This library is an implementation of the Mustache template language in Go.

### Mustache Spec Compliance

https://github.com/mustache/spec contains the formal standard for Mustache, and it is added as a submodule (using v1.1.3) for testing compliance. Currently ~40% of tests are failing, and the optional lambda support is not implemented. You can see which tests are disabled (b/c they are failing) by looking at spec_test.go. Getting all tests passing is my top priority (time permitting), and any PRs to that end are welcome.

## Documentation

For more information about mustache, check out the [mustache project page](http://github.com/defunkt/mustache) or the [mustache manual](http://mustache.github.com/mustache.5.html).

Also check out some [example mustache files](http://github.com/defunkt/mustache/tree/master/examples/)

## Installation
To install mustache.go, simply run `go get github.com/cbroglie/mustache`. To use it in a program, use `import "github.com/cbroglie/mustache"`

## Usage
There are four main methods in this package:

```go
Render(data string, context ...interface{}) (string, error)

RenderFile(filename string, context ...interface{}) (string, error)

ParseString(data string) (*Template, error)

ParseFile(filename string) (*Template, error)
```

There are also two additional methods for using layouts (explained below).

The Render method takes a string and a data source, which is generally a map or struct, and returns the output string. If the template file contains an error, the return value is a description of the error. There's a similar method, RenderFile, which takes a filename as an argument and uses that for the template contents.

```go
data, err := mustache.Render("hello {{c}}", map[string]string{"c": "world"})
```

If you're planning to render the same template multiple times, you do it efficiently by compiling the template first:

```go
tmpl, _ := mustache.ParseString("hello {{c}}")
var buf bytes.Buffer
for i := 0; i < 10; i++ {
    tmpl.FRender(&buf, map[string]string{"c": "world"})
}
```

For more example usage, please see `mustache_test.go`

## Escaping

mustache.go follows the official mustache HTML escaping rules. That is, if you enclose a variable with two curly brackets, `{{var}}`, the contents are HTML-escaped. For instance, strings like `5 > 2` are converted to `5 &gt; 2`. To use raw characters, use three curly brackets `{{{var}}}`.

## Layouts

It is a common pattern to include a template file as a "wrapper" for other templates. The wrapper may include a header and a footer, for instance. Mustache.go supports this pattern with the following two methods:

```go
RenderInLayout(data string, layout string, context ...interface{}) (string, error)

RenderFileInLayout(filename string, layoutFile string, context ...interface{}) (string, error)
```

The layout file must have a variable called `{{content}}`. For example, given the following files:

layout.html.mustache:

```html
<html>
<head><title>Hi</title></head>
<body>
{{{content}}}
</body>
</html>
```

template.html.mustache:

```html
<h1>Hello World!</h1>
```

A call to `RenderFileInLayout("template.html.mustache", "layout.html.mustache", nil)` will produce:

```html
<html>
<head><title>Hi</title></head>
<body>
<h1>Hello World!</h1>
</body>
</html>
```

## A note about method receivers

Mustache.go supports calling methods on objects, but you have to be aware of Go's limitations. For example, lets's say you have the following type:

```go
type Person struct {
    FirstName string
    LastName string    
}

func (p *Person) Name1() string {
    return p.FirstName + " " + p.LastName
}

func (p Person) Name2() string {
    return p.FirstName + " " + p.LastName
}
```

While they appear to be identical methods, `Name1` has a pointer receiver, and `Name2` has a value receiver. Objects of type `Person`(non-pointer) can only access `Name2`, while objects of type `*Person`(person) can access both. This is by design in the Go language.

So if you write the following:

```go
mustache.Render("{{Name1}}", Person{"John", "Smith"})
```

It'll be blank. You either have to use `&Person{"John", "Smith"}`, or call `Name2`

## Supported features

* Variables
* Comments
* Change delimiter
* Sections (boolean, enumerable, and inverted)
* Partials
