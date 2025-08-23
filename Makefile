.PHONY: dev build build-prod lint test test-snapshots test-snapshots-update version version-major version-minor version-patch

dev:
	go run main.go

build:
	go build -o tmp/symbolista

build-prod:
	CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o tmp/symbolista

lint:
	go vet ./... && go fmt ./...

test:
	go test ./...
	cd tests && go test -v

test-snapshots:
	cd tests && go test -v

test-snapshots-update:
	cd tests && UPDATE_SNAPSHOTS=1 go test -v

# Versioning commands
version:
	@echo "Usage: make version-<level> where level is major, minor, or patch"
	@echo "Examples:"
	@echo "  make version-patch  # 1.0.0 -> 1.0.1"
	@echo "  make version-minor  # 1.0.1 -> 1.1.0"
	@echo "  make version-major  # 1.1.0 -> 2.0.0"
	@echo ""
	@echo "Current version:"
	@git describe --tags --abbrev=0 2>/dev/null || echo "No tags found (will start from v0.0.0)"

version-major:
	./scripts/bump-version.sh major

version-minor:
	./scripts/bump-version.sh minor

version-patch:
	./scripts/bump-version.sh patch
