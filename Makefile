BINARY  := gp-cli
MODULE  := github.com/area99/patent-cli
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build install clean test tidy cross

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/gp-cli/

install:
	go install $(LDFLAGS) ./cmd/gp-cli/

clean:
	rm -f $(BINARY)
	rm -rf dist/

test:
	go test ./...

tidy:
	go mod tidy

# Cross-compile for common platforms
cross:
	mkdir -p dist
	GOOS=darwin  GOARCH=arm64  go build $(LDFLAGS) -o dist/$(BINARY)-darwin-arm64   ./cmd/gp-cli/
	GOOS=darwin  GOARCH=amd64  go build $(LDFLAGS) -o dist/$(BINARY)-darwin-amd64   ./cmd/gp-cli/
	GOOS=linux   GOARCH=amd64  go build $(LDFLAGS) -o dist/$(BINARY)-linux-amd64    ./cmd/gp-cli/
	GOOS=linux   GOARCH=arm64  go build $(LDFLAGS) -o dist/$(BINARY)-linux-arm64    ./cmd/gp-cli/
	GOOS=windows GOARCH=amd64  go build $(LDFLAGS) -o dist/$(BINARY)-windows-amd64.exe ./cmd/gp-cli/
	@echo "Built:"
	@ls -lh dist/
