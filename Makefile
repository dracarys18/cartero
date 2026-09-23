.PHONY: all build clean run test install deps fmt vet lint

all: build

deps:
	go mod download
	go mod tidy

build:
	go build -o bin/cartero ./cmd/cartero

install:
	go install ./cmd/cartero

run: build
	./bin/cartero

clean:
	rm -rf bin/
	rm -rf output/
	rm -f *.db

test:
	go test -v ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

lint: fmt vet

help:
	@echo "Available targets:"
	@echo "  build     - Build the bot binary"
	@echo "  run       - Run the bot with config.toml"
	@echo "  clean     - Clean build artifacts and databases"
	@echo "  deps      - Download dependencies"
	@echo "  test      - Run tests"
	@echo "  fmt       - Format code"
	@echo "  vet       - Run go vet"
	@echo "  lint      - Run fmt and vet"
