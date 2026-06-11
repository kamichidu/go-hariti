.PHONY: all
all: test build

.PHONY: build
build: generate
	go build -ldflags "-X main.appVersion=$$(git describe --tags)" ./cmd/hariti/

.PHONY: generate
generate:
	go generate ./...

.PHONY: debug
debug: build
	./hariti ${ARGS}

.PHONY: test
test: generate
	go test ${VERBOSE} ./...

.PHONY: deps
deps:
	go mod download
