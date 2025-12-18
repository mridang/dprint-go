.PHONY: default build build-gofmt build-shfmt build-tffmt build-cuefmt build-protofmt lint test test-gofmt test-shfmt test-tffmt test-cuefmt test-protofmt vendor clean format

export GO111MODULE=on

ROOT_BUILD_DIR := $(abspath build)

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
test: test-gofmt test-shfmt test-tffmt test-cuefmt test-protofmt

# Run tests only in gofmt command package
test-gofmt:
	GOFLAGS= CGO_ENABLED=1 go test -mod=mod -v=true -cover=true -count=1 ./cmd/gofmt

# Run tests only in shfmt command package
test-shfmt:
	GOFLAGS= CGO_ENABLED=1 go test -mod=mod -v=true -cover=true -count=1 ./cmd/shfmt

# Run tests only in tffmt command package
test-tffmt:
	GOFLAGS= CGO_ENABLED=1 go test -mod=mod -v=true -cover=true -count=1 ./cmd/tffmt

# Run tests only in cuefmt command package
test-cuefmt:
	GOFLAGS= CGO_ENABLED=1 go test -mod=mod -v=true -cover=true -count=1 ./cmd/cuefmt

# Run tests only in protofmt command package
test-protofmt:
	GOFLAGS= CGO_ENABLED=1 go test -mod=mod -v=true -cover=true -count=1 ./cmd/protofmt

vendor:
	go mod vendor

clean:
	rm -rf ./build

format:
	go install golang.org/x/tools/cmd/goimports@latest
	goimports -w=true ./
	gofmt -s=true -w=true ./
