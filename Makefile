SHELL := /bin/bash

.PHONY: schema build test cover bench bench-smoke fuzz fuzz-smoke lint vuln docs docs-check docs-serve docs-build release-check integration check clean

BINARY   := gh-fleet
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE     ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  := -s -w -X github.com/rossedman/gh-fleet/cmd.version=$(VERSION) -X github.com/rossedman/gh-fleet/cmd.commit=$(COMMIT) -X github.com/rossedman/gh-fleet/cmd.date=$(DATE)

# Pinned so local runs and CI use the same tools; `go run` fetches them and
# builds them with this module's toolchain, so they understand its Go version.
GOVERSION   := $(shell go env GOVERSION)
GORUN       := GOTOOLCHAIN=$(GOVERSION) go run
GOLANGCI    := $(GORUN) github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0
GOVULNCHECK := $(GORUN) golang.org/x/vuln/cmd/govulncheck@v1.9.0
GORELEASER  := $(GORUN) github.com/goreleaser/goreleaser/v2@v2.19.0
MKDOCS      := uv run --no-project --with-requirements requirements-docs.txt mkdocs

COVER_MIN ?= 90
FUZZTIME  ?= 30s
COVERPKG  := ./cmd/...,./internal/...,./tools/...

build:
	go build -trimpath -ldflags '$(LDFLAGS)' -o $(BINARY) .

test:
	go test -race ./...

# Coverage of hand-written code only: generated files and examples/ are
# filtered out of the profile before totaling.
cover:
	go test -race -coverpkg=$(COVERPKG) -coverprofile=coverage.out ./...
	@{ head -1 coverage.out; grep -v '^mode:' coverage.out | { grep -vFf <(go run ./tools/generated ./... ; echo '^$$') || true; } | grep -v '/examples/'; } > coverage.hand.out
	@go tool cover -func=coverage.hand.out | awk -v min=$(COVER_MIN) '/^total:/ {sub("%","",$$3); print "coverage " $$3 "%"; if ($$3+0 < min) {print "coverage below " min "%"; exit 1}}'

bench:
	go test -run='^$$' -bench=. -benchmem -count=6 ./... | tee bench/$(shell date +%Y%m%d-%H%M%S).txt

bench-smoke:
	go test -run='^$$' -bench=. -benchtime=1x ./...

# Runs every Fuzz target for FUZZTIME.
fuzz:
	@for pkg in $$(go list ./...); do \
	  for f in $$(go test -list='^Fuzz' $$pkg | grep '^Fuzz'); do \
	    echo "fuzz $$pkg $$f"; \
	    go test -run='^$$' -fuzz="^$$f$$" -fuzztime=$(FUZZTIME) $$pkg || exit 1; \
	  done; \
	done

# shellcheck is preinstalled on GitHub's Ubuntu runners; locally, install it
# or run `make lint SHELLCHECK="uvx --from shellcheck-py shellcheck"`.
SHELLCHECK ?= shellcheck

lint:
	gofmt -l . | (! grep .)
	$(SHELLCHECK) tools/*.sh
	$(GOLANGCI) run ./...
	./tools/check-cmds.sh
	go run ./tools/doccheck ./...

vuln:
	$(GOVULNCHECK) ./...

docs:
	go run ./tools/gendocs

# Refreshes GitHub's GraphQL schema, which the ghapi tests validate every
# query against.
schema:
	curl -fsSL https://docs.github.com/public/fpt/schema.docs.graphql | gzip -9n > internal/ghapi/testdata/schema.docs.graphql.gz

# Fails when docs/reference/ or the generated mkdocs.yml nav is stale.
docs-check: docs
	git diff --exit-code -- docs/reference mkdocs.yml
	@test -z "$$(git ls-files --others --exclude-standard -- docs/reference)" || { echo "untracked generated docs:"; git ls-files --others --exclude-standard -- docs/reference; exit 1; }

docs-serve: docs
	$(MKDOCS) serve

docs-build: docs
	$(MKDOCS) build --strict

release-check:
	$(GORELEASER) check
	$(GORELEASER) release --snapshot --clean --skip=sign

# Opt-in: needs GH_FLEET_TEST_ORG and GH_FLEET_TEST_REPO and a gh login.
integration:
	go test -tags integration -run Integration -count=1 -v ./internal/app/

check: build lint cover docs-check bench-smoke fuzz-smoke vuln docs-build release-check

fuzz-smoke:
	$(MAKE) fuzz FUZZTIME=10s

clean:
	rm -rf $(BINARY) dist site coverage.out coverage.hand.out
