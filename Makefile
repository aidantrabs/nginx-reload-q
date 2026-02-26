BINARY=nginx-reload-q
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build clean run test

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/server/

clean:
	rm -f $(BINARY)

run: build
	sudo ./$(BINARY)

test:
	go test ./... -race
