.PHONY: fmt check test build package

DIST_DIR ?= zig-out/package

fmt:
	zig fmt build.zig core
	cd tui && /usr/local/go/bin/gofmt -w $$(find . -name '*.go')

check:
	zig fmt --check build.zig core
	cd tui && /usr/local/go/bin/go vet ./...

test:
	zig build test
	cd tui && /usr/local/go/bin/go test ./...

build:
	zig build
	cd tui && /usr/local/go/bin/go build -o ../zig-out/bin/zintent ./cmd/zintent

package: build
	./scripts/package.sh "$(DIST_DIR)"
