APP := palbot

.PHONY: build test run tidy

build:
	go build -o bin/$(APP) ./cmd/palbot

test:
	go test ./...

run:
	go run ./cmd/palbot

tidy:
	go mod tidy
