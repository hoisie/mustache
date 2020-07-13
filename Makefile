export GOFLAGS := -mod=vendor

.PHONY: all
all: bin/mustache

.PHONY: clean
clean:
	rm -rf bin

.PHONY: ci
ci: fmt lint test

.PHONY: test
test:
	go test -race -coverprofile=coverage.txt -covermode=atomic ./...

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: lint
lint: bin/golangci-lint
	./bin/golangci-lint run ./...

SOURCES     := $(shell find . -name '*.go')
BUILD_FLAGS ?= -v
LDFLAGS     ?= -w -s

bin/golangci-lint: $(SOURCES)
	go build -o bin/golangci-lint ./vendor/github.com/golangci/golangci-lint/cmd/golangci-lint

bin/%: $(SOURCES)
	CGO_ENABLED=0 go build -o $@ $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" ./cmd/$(@F)
