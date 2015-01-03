.PHONY: generate format test

GOFMT=gofmt -s

all: format test

format:
	${GOFMT} -w *.go

test:
	go test

generate:
	go generate
	${GOFMT} -w mustache_spec_test.go
