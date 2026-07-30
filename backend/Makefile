.PHONY: run build test fmt lint tidy

run:
	go run ./cmd/api

build:
	CGO_ENABLED=0 go build -o bin/api ./cmd/api

test:
	go test ./...

fmt:
	gofmt -l -w .

lint:
	go vet ./...
	@test -z "$$(gofmt -l .)" || (echo "gofmt: files need formatting:" && gofmt -l . && exit 1)

tidy:
	go mod tidy
