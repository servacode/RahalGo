.PHONY: run build test fmt lint tidy seed

run:
	go run ./cmd/api

# زراعة بيانات التطوير (حسابات بكلمات مرور معروفة، متاجر، مناطق، أكواد)
seed:
	go run ./cmd/seed

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
