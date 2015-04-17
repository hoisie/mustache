.PHONY: generate format test

GOFMT=goimports

IMPORT_BASE := github.com/jabley
IMPORT_PATH := $(IMPORT_BASE)/mustache

all: _vendor _deps format test

format:
	${GOFMT} -w *.go
	${GOFMT} -w ./parse/*.go

test:
	gom test -v . ./parse 

coverage:
	gom test -coverprofile=mustache.coverprofile
	find . -name '*.coverprofile' -type f -exec sed -i '' 's|_'$(CURDIR)'|\.|' {} \;

generate:
	gom generate
	${GOFMT} -w mustache_spec_test.go

_deps:
	go get github.com/mattn/gom

_vendor: Gomfile _vendor/src/$(IMPORT_PATH)
	gom -test install
	touch _vendor

_vendor/src/$(IMPORT_PATH):
	rm -f _vendor/src/$(IMPORT_PATH)
	mkdir -p _vendor/src/$(IMPORT_BASE)
	ln -s $(CURDIR) _vendor/src/$(IMPORT_PATH)
