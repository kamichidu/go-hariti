.PHONY: all
all: test build

.PHONY: build
build: generate
	go build -ldflags "-X main.appVersion=$$(git describe --tags)" ./cmd/hariti/

.PHONY: generate
generate:
	go generate -v $$(glide novendor)

.PHONY: test
test: generate
	go test ${VERBOSE} $$(glide novendor)

.PHONY: deps
deps:
	go get -v golang.org/x/tools/cmd/goyacc
	go get -v github.com/Masterminds/glide
	glide install
