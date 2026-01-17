# DevLog CLI Makefile

BINARY_NAME=devlog
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X github.com/leffen/devlog-cli/cmd.Version=$(VERSION) -X github.com/leffen/devlog-cli/cmd.Commit=$(COMMIT)"

.PHONY: all build clean install test fmt lint release release-dry snapshot

all: build

# Build the binary
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

# Build for all platforms
build-all:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 .
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe .

# Install to GOPATH/bin
install:
	go install $(LDFLAGS) .

# Install to /usr/local/bin
install-local: build
	cp $(BINARY_NAME) /usr/local/bin/

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -rf dist/

# Run tests
test:
	go test -v ./...

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Update dependencies
deps:
	go mod tidy
	go mod download

# Show version
version:
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"

# GoReleaser targets
# Create a snapshot release (no publish)
snapshot:
	goreleaser release --snapshot --clean

# Dry run of release (validates config)
release-dry:
	goreleaser release --skip=publish --clean

# Create and publish a release (requires GITHUB_TOKEN)
release:
	goreleaser release --clean

# Check goreleaser config
check:
	goreleaser check
