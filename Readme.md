## Overview

mustache.go is an implementation of the mustache template language in Go. It is better suited for website templates than Go's native pkg/template. mustache.go is fast -- it parses templates efficiently and stores them in a tree-like structure which allows for fast execution. 

## Documentation

For more information about mustache, check out the [mustache project page](http://github.com/defunkt/mustache) or the [mustache manual](http://mustache.github.com/mustache.5.html).

Also check out some [example mustache files](http://github.com/defunkt/mustache/tree/master/examples/)

## Usage

There are only four methods in this package:

    func Render(data string, context ...interface{}) string
    
    func RenderFile(filename string, context ...interface{}) string
    
    func ParseString(data string) (*template, os.Error)
    
    func ParseFile(filename string) (*template, os.Error) 


The Render method takes a string and a data source, which is generally a map or struct, and returns the output string. If the template file contains an error, the return value is a description of the error. There's a similar method, RenderFile, which takes a filename as an argument and uses that for the template contents. 

    data := mustache.Render("hello {{c}}", map[string]string{"c":"world"})
    println(data)


If you're planning to render the same template multiple times, you do it efficiently by compiling the template first:

    tmpl,_ := mustache.Parse("hello {{c}}")
    var buf bytes.Buffer;
    for i := 0; i < 10; i++ {
        tmpl.Render (map[string]string { "c":"world"}, &buf)  
    }

For more example usage, please see `mustache_test.go`

## Escaping

mustache.go follows the official mustache HTML escaping rules. That is, if you enclose a variable with two curly brackets, `{{var}}`, the contents are HTML-escaped. For instance, strings like `5 > 2` are converted to `5 &gt; 2`. To use raw characters, use three curly brackets `{{{var}}}`.
 
## Supported features

* Variables
* Comments
* Change delimiter
* Sections (boolean, enumerable, and inverted)
* Partials


