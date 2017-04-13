all: test

build: generate
	go build -ldflags "-X main.appVersion=$$(git describe --tags)" ./cmd/hariti/

generate:
	go generate -v $$(glide novendor)

test: generate
	go test -v $$(glide novendor)

deps:
	go get -v golang.org/x/tools/cmd/goyacc
