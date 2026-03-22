BINARY     := skills
CMD        := ./cmd/skills
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS    := -ldflags "-X main.version=$(VERSION)"

.PHONY: build clean install test lint

build:
	go build $(LDFLAGS) -o $(BINARY) $(CMD)

install: build
	install -m 755 $(BINARY) /usr/local/bin/$(BINARY)

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -f $(BINARY)

# Cross-compile for macOS (both architectures)
release-darwin:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-amd64 $(CMD)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-arm64 $(CMD)
	GOOS=linux  GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-amd64  $(CMD)
