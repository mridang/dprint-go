.PHONY: default build build-gofmt build-shfmt lint test test-gofmt test-shfmt vendor clean format

export GO111MODULE=on

default: build

build: build-gofmt build-shfmt

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

lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run --verbose

# Force module mode and CGO so wasmer-go finds its packaged libs.
test: test-gofmt test-shfmt

# Run tests only in gofmt command package
test-gofmt:
	GOFLAGS= CGO_ENABLED=1 go test -mod=mod -v=true -cover=true -count=1 ./cmd/gofmt

# Run tests only in shfmt command package
test-shfmt:
	GOFLAGS= CGO_ENABLED=1 go test -mod=mod -v=true -cover=true -count=1 ./cmd/shfmt

vendor:
	go mod vendor

clean:
	rm -rf ./build

format:
	go install golang.org/x/tools/cmd/goimports@latest
	goimports -w=true ./
	gofmt -s=true -w=true ./
