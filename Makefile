.PHONY: build run clean install test lint fmt help

# Binary name
BINARY=hotbrew
VERSION?=0.1.0

# Build flags
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION)"

## build: Build the binary
build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/hotbrew

## run: Build and run
run: build
	./$(BINARY)

## install: Install to $GOPATH/bin
install:
	go install $(LDFLAGS) ./cmd/hotbrew

## clean: Remove build artifacts
clean:
	rm -f $(BINARY)
	rm -rf dist/

## test: Run tests
test:
	go test -v ./...

## lint: Run linter
lint:
	golangci-lint run

## fmt: Format code
fmt:
	go fmt ./...
	goimports -w .

## deps: Download dependencies
deps:
	go mod tidy
	go mod download

## release: Build for multiple platforms
release: clean
	mkdir -p dist
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-amd64 ./cmd/hotbrew
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-arm64 ./cmd/hotbrew
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-amd64 ./cmd/hotbrew
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-arm64 ./cmd/hotbrew
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-windows-amd64.exe ./cmd/hotbrew

## help: Show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
