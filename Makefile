build:
	CGO_ENABLED=0 go build -o strongbox ./cmd/strongbox/

run: build
	./strongbox

test:
	go test ./...

clean:
	rm -f strongbox

.PHONY: build run test clean
