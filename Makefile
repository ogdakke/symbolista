.PHONY: dev build lint test

dev:
	go run main.go

build:
	go build -o tmp/symbolista

lint:
	go vet ./... && go fmt ./...

test:
	go test ./...
