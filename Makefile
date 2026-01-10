.PHONY: default build build-gofmt build-shfmt build-tffmt build-cuefmt build-protofmt lint test test-gofmt test-shfmt test-tffmt test-cuefmt test-protofmt vendor clean format clean-coverage ensure-gotestsum

export GO111MODULE=on

ROOT_BUILD_DIR := $(abspath build)
GOTESTSUM ?= gotestsum
TEST_FORMAT ?= pkgname-and-test-fails
COVER_DIR := .out/cover
JUNIT_FILE := .out/junit.xml
LCOV_FILE := .out/lcov.info

default: build

build: build-gofmt build-shfmt build-tffmt build-cuefmt build-protofmt

build-gofmt:
	$(MAKE) -C cmd/gofmt build OUT_DIR=$(ROOT_BUILD_DIR)

build-shfmt:
	$(MAKE) -C cmd/shfmt build OUT_DIR=$(ROOT_BUILD_DIR)

build-tffmt:
	$(MAKE) -C cmd/tffmt build OUT_DIR=$(ROOT_BUILD_DIR)

build-cuefmt:
	$(MAKE) -C cmd/cuefmt build OUT_DIR=$(ROOT_BUILD_DIR)

build-protofmt:
	$(MAKE) -C cmd/protofmt build OUT_DIR=$(ROOT_BUILD_DIR)

lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run --verbose

# Force module mode and CGO so wasmer-go finds its packaged libs.
test: clean-coverage ensure-gotestsum
	mkdir -p $(GOCACHE) $(GOTMPDIR) $(dir $(LCOV_FILE))
	GOFLAGS= CGO_ENABLED=1 GOCACHE=$(GOCACHE) GOTMPDIR=$(GOTMPDIR) $(GOTESTSUM) --junitfile $(JUNIT_FILE) --format $(TEST_FORMAT) -- -mod=mod -coverpkg=./... -covermode=atomic -coverprofile=$(LCOV_FILE) -count=1 ./...

# Run tests only in gofmt command package
test-gofmt:
	GOFLAGS= CGO_ENABLED=1 $(GOTESTSUM) --format $(TEST_FORMAT) -- -mod=mod -coverpkg=./cmd/gofmt -covermode=atomic -count=1 ./cmd/gofmt

# Run tests only in shfmt command package
test-shfmt:
	GOFLAGS= CGO_ENABLED=1 $(GOTESTSUM) --format $(TEST_FORMAT) -- -mod=mod -coverpkg=./cmd/shfmt -covermode=atomic -count=1 ./cmd/shfmt

# Run tests only in tffmt command package
test-tffmt:
	GOFLAGS= CGO_ENABLED=1 $(GOTESTSUM) --format $(TEST_FORMAT) -- -mod=mod -coverpkg=./cmd/tffmt -covermode=atomic -count=1 ./cmd/tffmt

# Run tests only in cuefmt command package
test-cuefmt:
	GOFLAGS= CGO_ENABLED=1 $(GOTESTSUM) --format $(TEST_FORMAT) -- -mod=mod -coverpkg=./cmd/cuefmt -covermode=atomic -count=1 ./cmd/cuefmt

# Run tests only in protofmt command package
test-protofmt:
	GOFLAGS= CGO_ENABLED=1 $(GOTESTSUM) --format $(TEST_FORMAT) -- -mod=mod -coverpkg=./cmd/protofmt -covermode=atomic -count=1 ./cmd/protofmt

clean-coverage:
	rm -rf .out

vendor:
	go mod vendor

clean:
	rm -rf ./build

format:
	go install golang.org/x/tools/cmd/goimports@latest
	goimports -w=true ./
	gofmt -s=true -w=true ./

ensure-gotestsum:
	@if ! command -v $(GOTESTSUM) >/dev/null 2>&1; then \
		GOFLAGS=-mod=mod go install gotest.tools/gotestsum@latest; \
	fi
