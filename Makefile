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
lint: bin/golint
	go list ./... | xargs -L1 ./bin/golint -set_exit_status

bin/golint: $(shell find . -type f -name '*.go')
	@mkdir -p $(dir $@)
	go build -o $@ ./vendor/golang.org/x/lint/golint

bin/%: $(shell find . -type f -name '*.go')
	@mkdir -p $(dir $@)
	go build -o $@ ./cmd/$(@F)
