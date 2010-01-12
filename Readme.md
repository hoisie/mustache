### Note ###

This project recently changed its name from gostache to mustache.go. If you have a local copy, you'll need to modify the .git/config file in the project directory and change the 'url' property to:
    
    git://github.com/hoisie/mustache.go.git

## Overview

mustache.go is an implementation of the mustache template language in Go. It is better suited for website templates than Go's native pkg/template. mustache.go is fast -- it parses templates efficiently and stores them in a tree-like structure which allows for fast execution. 

For more information about mustache, check out the [mustache project page] ( http://github.com/defunkt/mustache ).

## Supported features

* Variables
* Comments
* Change delimiter
* Sections
* Partials


