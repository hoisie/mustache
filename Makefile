.PHONY: generate format test

GOFMT=gofmt -s

all: format test

format:
	${GOFMT} -w *.go

test:
	go test

coverage:
	go test -coverprofile=mustache.coverprofile
	find . -name '*.coverprofile' -type f -exec sed -i '' 's|_'$(CURDIR)'|\.|' {} \;

generate:
	go generate
	${GOFMT} -w mustache_spec_test.go
