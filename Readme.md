### Note

This project recently changed its name from gostache to mustache.go. If you have a local copy, you'll need to modify the .git/config file in the project directory and change the 'url' property to:
    
    url = git://github.com/hoisie/mustache.go.git

## Overview

mustache.go is an implementation of the mustache template language in Go. It is better suited for website templates than Go's native pkg/template. mustache.go is fast -- it parses templates efficiently and stores them in a tree-like structure which allows for fast execution. 

For more information about mustache, check out the [mustache project page] ( http://github.com/defunkt/mustache ).

## Usage

The Render method takes a string and a data source, which is either a map or struct. There's an analagous method, RenderFile, which takes a filename as an argument and uses that for the template contents. 

    data,_ := mustache.Render("hello {{c}}", map[string]string{"c":"world"})
    println(data)


If you're planning to render the same template multiple times, you do it efficiently by compiling the template first:

    tmpl,_ := mustache.Parse("hello {{c}}")
    var buf bytes.Buffer;
    tmpl.Render (map[string]string { "c":"world"}, &buf)  

## Supported features

* Variables
* Comments
* Change delimiter
* Sections
* Partials


