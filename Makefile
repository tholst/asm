BINARY     := skills
CMD        := ./cmd/skills
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS    := -ldflags "-X main.version=$(VERSION)"

.PHONY: build clean install test lint release-patch release-minor release-major _check-clean

build:
	go build $(LDFLAGS) -o $(BINARY) $(CMD)

install:
	go install $(LDFLAGS) $(CMD)

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -f $(BINARY)

_check-clean:
	@git diff --quiet && git diff --cached --quiet || (echo "Error: uncommitted changes — commit or stash first"; exit 1)

release-patch: _check-clean
	@current=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	major=$$(echo $$current | sed 's/v//' | cut -d. -f1); \
	minor=$$(echo $$current | sed 's/v//' | cut -d. -f2); \
	patch=$$(echo $$current | sed 's/v//' | cut -d. -f3); \
	new="v$$major.$$minor.$$((patch+1))"; \
	echo "Tagging $$new"; \
	git tag $$new && git push origin $$new

release-minor: _check-clean
	@current=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	major=$$(echo $$current | sed 's/v//' | cut -d. -f1); \
	minor=$$(echo $$current | sed 's/v//' | cut -d. -f2); \
	new="v$$major.$$((minor+1)).0"; \
	echo "Tagging $$new"; \
	git tag $$new && git push origin $$new

release-major: _check-clean
	@current=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	major=$$(echo $$current | sed 's/v//' | cut -d. -f1); \
	new="v$$((major+1)).0.0"; \
	echo "Tagging $$new"; \
	git tag $$new && git push origin $$new

# Cross-compile for macOS (both architectures)
release-darwin:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-amd64 $(CMD)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-arm64 $(CMD)
	GOOS=linux  GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-amd64  $(CMD)
