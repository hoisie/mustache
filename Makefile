include $(GOROOT)/src/Make.inc

TARG=mustache
GOFILES=\
	mustache.go\

include $(GOROOT)/src/Make.pkg

format:
	gofmt -spaces=true -tabindent=false -tabwidth=4 -w mustache.go
	gofmt -spaces=true -tabindent=false -tabwidth=4 -w mustache_test.go

