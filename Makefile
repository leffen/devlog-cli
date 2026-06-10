# DevLog CLI Makefile

BINARY_NAME=devlog
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS=-ldflags "-s -w -X github.com/leffen/devlog-cli/internal/cmd.Version=$(VERSION) -X github.com/leffen/devlog-cli/internal/cmd.Commit=$(COMMIT)"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

.PHONY: all build build-all clean install install-local test test-verbose test-coverage \
        fmt lint vet deps version snapshot release-dry release release-local check help \
        env-encrypt env-decrypt

# Default target
all: build

## Build targets

# Build the binary
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .

# Build from cmd/devlog (for go install compatibility)
build-cmd:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) ./cmd/devlog

# Build for all platforms
build-all: clean
	@mkdir -p dist
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 .
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 .
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe .

## Install targets

# Install to GOPATH/bin
install:
	$(GOCMD) install $(LDFLAGS) ./cmd/devlog

# Install to /usr/local/bin
install-local: build
	cp $(BINARY_NAME) /usr/local/bin/

# Uninstall from /usr/local/bin
uninstall-local:
	rm -f /usr/local/bin/$(BINARY_NAME)

## Test targets

# Run all tests
test:
	$(GOTEST) ./...

# Run tests with verbose output
test-verbose:
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with race detection
test-race:
	$(GOTEST) -race ./...

# Run a specific test
test-run:
	@test -n "$(TEST)" || (echo "Usage: make test-run TEST=TestName" && exit 1)
	$(GOTEST) -v -run $(TEST) ./...

## Code quality targets

# Format code
fmt:
	$(GOFMT) ./...

# Vet code
vet:
	$(GOVET) ./...

# Lint code (requires golangci-lint)
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run

# Run all checks (fmt, vet, lint, test)
check-all: fmt vet lint test

## Dependency targets

# Update dependencies
deps:
	$(GOMOD) tidy
	$(GOMOD) download

# Update dependencies to latest versions
deps-update:
	$(GOMOD) tidy
	$(GOCMD) get -u ./...
	$(GOMOD) tidy

# Verify dependencies
deps-verify:
	$(GOMOD) verify

## Clean targets

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -rf dist/
	rm -f coverage.out coverage.html

## Release targets (GoReleaser)

# Create a snapshot release (no publish)
snapshot:
	goreleaser release --snapshot --clean

# Dry run of release (validates config)
release-dry:
	goreleaser release --skip=publish --clean

# Which version component to bump for `make release` (major | minor | patch).
BUMP ?= minor

# Cut a release: bump the latest v* tag, then create and push the new tag.
# Pushing the tag triggers the GitHub Actions release workflow, which runs
# GoReleaser to build and publish. Override the component with BUMP=patch etc.
release:
	@latest=$$(git tag --list 'v*' --sort=-v:refname | head -n1); \
	latest=$${latest:-v0.0.0}; \
	ver=$${latest#v}; \
	major=$$(echo "$$ver" | cut -d. -f1); \
	minor=$$(echo "$$ver" | cut -d. -f2); \
	patch=$$(echo "$$ver" | cut -d. -f3); \
	case "$(BUMP)" in \
		major) major=$$((major+1)); minor=0; patch=0;; \
		minor) minor=$$((minor+1)); patch=0;; \
		patch) patch=$$((patch+1));; \
		*) echo "Error: BUMP must be major, minor, or patch (got '$(BUMP)')." >&2; exit 1;; \
	esac; \
	next="v$$major.$$minor.$$patch"; \
	if [ -n "$$(git tag --list "$$next")" ]; then \
		echo "Error: tag $$next already exists." >&2; exit 1; \
	fi; \
	if [ -n "$$(git status --porcelain)" ]; then \
		echo "Error: working tree is dirty; commit or stash changes first." >&2; exit 1; \
	fi; \
	echo "Releasing $$latest -> $$next ($(BUMP) bump)"; \
	git tag -a "$$next" -m "Release $$next" && \
	git push origin "$$next" && \
	echo "Pushed $$next — GitHub Actions will build and publish the release."

# Publish a release from the current machine instead of via CI.
# Uses GITHUB_TOKEN / HOMEBREW_TAP_TOKEN from the environment if set, otherwise
# falls back to the logged-in `gh` CLI token. Both the release repo and the
# homebrew tap repo are owned by the same account, so one personal token serves
# both. The token is resolved at runtime inside the shell (with @ to suppress
# echo) so it is never printed to the terminal. Requires an existing tag.
release-local:
	@token="$${GITHUB_TOKEN:-$$(gh auth token 2>/dev/null)}"; \
	if [ -z "$$token" ]; then \
		echo "Error: no GitHub token found. Set GITHUB_TOKEN or run 'gh auth login -s repo'." >&2; \
		exit 1; \
	fi; \
	GITHUB_TOKEN="$$token" HOMEBREW_TAP_TOKEN="$${HOMEBREW_TAP_TOKEN:-$$token}" goreleaser release --clean

# Check goreleaser config
check:
	goreleaser check

## Secrets (age-encrypted .env)

# Location of the age identities used for encrypt/decrypt. Override on the
# command line if your keys live elsewhere: make env-encrypt AGE_KEYS=/path/keys.txt
AGE_KEYS ?= $(HOME)/.config/sops/age/keys.txt

# Encrypt .env -> .env.enc for every age public key in the identity file, so any
# of your keys can decrypt it. Commit .env.enc; the plaintext .env stays ignored.
env-encrypt:
	@test -f .env || { echo "Error: no .env file to encrypt." >&2; exit 1; }
	@test -f "$(AGE_KEYS)" || { echo "Error: age keys file not found: $(AGE_KEYS)" >&2; exit 1; }
	@recipients=$$(grep -E '^# public key:' "$(AGE_KEYS)" | sed -E 's/^# public key: *//' | sed 's/^/-r /' | tr '\n' ' '); \
	test -n "$$recipients" || { echo "Error: no age public keys found in $(AGE_KEYS)" >&2; exit 1; }; \
	age -e $$recipients -o .env.enc .env && echo "Encrypted .env -> .env.enc"

# Decrypt .env.enc -> .env using your age identities.
env-decrypt:
	@test -f .env.enc || { echo "Error: no .env.enc file to decrypt." >&2; exit 1; }
	@test -f "$(AGE_KEYS)" || { echo "Error: age keys file not found: $(AGE_KEYS)" >&2; exit 1; }
	@age -d -i "$(AGE_KEYS)" -o .env .env.enc && echo "Decrypted .env.enc -> .env"

## Utility targets

# Show version info
version:
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"

# Run the CLI (for quick testing)
run:
	$(GOCMD) run . $(ARGS)

# Run with specific command
run-help:
	$(GOCMD) run . --help

run-version:
	$(GOCMD) run . --version

# Generate and display the CLAUDE.md prompt
prompt:
	$(GOCMD) run . prompt

## Help

# Show help
help:
	@echo "DevLog CLI Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Build targets:"
	@echo "  build        Build the binary"
	@echo "  build-cmd    Build from cmd/devlog"
	@echo "  build-all    Build for all platforms"
	@echo ""
	@echo "Install targets:"
	@echo "  install      Install to GOPATH/bin"
	@echo "  install-local Install to /usr/local/bin"
	@echo "  uninstall-local Remove from /usr/local/bin"
	@echo ""
	@echo "Test targets:"
	@echo "  test         Run all tests"
	@echo "  test-verbose Run tests with verbose output"
	@echo "  test-coverage Run tests with coverage report"
	@echo "  test-race    Run tests with race detection"
	@echo "  test-run     Run specific test (TEST=TestName)"
	@echo ""
	@echo "Code quality:"
	@echo "  fmt          Format code"
	@echo "  vet          Vet code"
	@echo "  lint         Lint code (requires golangci-lint)"
	@echo "  check-all    Run all checks (fmt, vet, lint, test)"
	@echo ""
	@echo "Dependencies:"
	@echo "  deps         Update and tidy dependencies"
	@echo "  deps-update  Update to latest versions"
	@echo "  deps-verify  Verify dependencies"
	@echo ""
	@echo "Release:"
	@echo "  snapshot     Create snapshot release"
	@echo "  release-dry  Dry run release"
	@echo "  release      Bump version (BUMP=minor) + push tag; CI publishes"
	@echo "  release-local Publish locally via goreleaser (uses gh token if env unset)"
	@echo "  check        Check goreleaser config"
	@echo ""
	@echo "Secrets:"
	@echo "  env-encrypt  Encrypt .env -> .env.enc (age)"
	@echo "  env-decrypt  Decrypt .env.enc -> .env (age)"
	@echo ""
	@echo "Utility:"
	@echo "  version      Show version info"
	@echo "  run          Run CLI with ARGS"
	@echo "  clean        Clean build artifacts"
	@echo "  help         Show this help"
