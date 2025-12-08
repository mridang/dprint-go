.PHONY: default build build-gofmt build-shfmt build-tffmt build-cuefmt build-protofmt lint test test-gofmt test-shfmt test-tffmt test-cuefmt test-protofmt vendor clean format patch-buf

export GO111MODULE=on

default: build

build: build-gofmt build-shfmt build-tffmt build-cuefmt build-protofmt

build-gofmt:
	mkdir -p build
	tinygo build -o=build/gofmt.wasm -target=wasm-unknown -scheduler=none -no-debug -opt=2 ./cmd/gofmt
	go run ./cmd/addstart/main.go build/gofmt.wasm build/gofmt-fixed.wasm
	mv build/gofmt-fixed.wasm build/gofmt.wasm

build-shfmt:
	mkdir -p build
	tinygo build -o=build/shfmt.wasm -target=wasm-unknown -scheduler=none -no-debug -opt=2 ./cmd/shfmt
	go run ./cmd/addstart/main.go build/shfmt.wasm build/shfmt-fixed.wasm
	mv build/shfmt-fixed.wasm build/shfmt.wasm

build-tffmt:
	mkdir -p build
	tinygo build -o=build/tffmt.wasm -target=wasm-unknown -scheduler=none -no-debug -opt=2 ./cmd/tffmt
	go run ./cmd/addstart/main.go build/tffmt.wasm build/tffmt-fixed.wasm
	mv build/tffmt-fixed.wasm build/tffmt.wasm

build-cuefmt:
	mkdir -p build
	# CUE is recursive; increase stack size to prevent "unreachable" (stack overflow) errors.
	tinygo build -o=build/cuefmt.wasm -target=wasm-unknown -scheduler=none -stack-size=128kb -no-debug -opt=1 ./cmd/cuefmt
	go run ./cmd/addstart/main.go build/cuefmt.wasm build/cuefmt-fixed.wasm
	mv build/cuefmt-fixed.wasm build/cuefmt.wasm

# Monkey patch the vendor directory to:
# 1. Disable the Buf private usage panic.
# 2. Remove 'go' keyword from thread.go to make execution synchronous (fixing "attempted to start a goroutine").
patch-buf:
	@echo "Monkey patching Buf..."
	@perl -pi -e 's/panic\("This code must only be imported/return; \/\/ panic\("This code must only be imported/g' vendor/github.com/bufbuild/buf/private/pkg/app/appproto/appproto.go || true
	@perl -pi -e 's/panic\("This code must only be imported/return; \/\/ panic\("This code must only be imported/g' vendor/buf.build/go/bufprivateusage/bufprivateusage.go || true
	@echo "Patching goroutines..."
	@find vendor/github.com/bufbuild/buf -name "thread.go" -exec perl -pi -e 's/go func/func/g' {} +

build-protofmt: patch-buf
	mkdir -p build
	# Use -scheduler=none for dprint compatibility
	tinygo build -o=build/protofmt.wasm -target=wasm-unknown -scheduler=none -no-debug -opt=2 ./cmd/protofmt
	go run ./cmd/addstart/main.go build/protofmt.wasm build/protofmt-fixed.wasm
	mv build/protofmt-fixed.wasm build/protofmt.wasm

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

# Run tests only in protofmt command package (depends on patch-buf)
test-protofmt: patch-buf
	GOFLAGS= CGO_ENABLED=1 go test -mod=vendor -v=true -cover=true -count=1 ./cmd/protofmt

vendor:
	go mod vendor

clean:
	rm -rf ./build

format:
	go install golang.org/x/tools/cmd/goimports@latest
	goimports -w=true ./
	gofmt -s=true -w=true ./
