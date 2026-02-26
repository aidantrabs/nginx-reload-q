BINARY=nginx-reload-q

.PHONY: build clean run

build:
	go build -o $(BINARY) ./cmd/server/

clean:
	rm -f $(BINARY)

run: build
	sudo ./$(BINARY)
