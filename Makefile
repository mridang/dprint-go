.PHONY: default build lint test vendor clean format

export GO111MODULE=on

default: build

build:
	mkdir -p build
	tinygo build -o=build/dprint.wasm -target=wasm-unknown -scheduler=none -no-debug -opt=2 main.go
	go run ./cmd/addstart/main.go build/dprint.wasm build/dprint-fixed.wasm
	mv build/dprint-fixed.wasm build/dprint.wasm

lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run --verbose

# Force module mode and CGO so wasmer-go finds its packaged libs.
test:
	GOFLAGS= CGO_ENABLED=1 go test -mod=mod -v=true -cover=true -count=1 ./...

vendor:
	go mod vendor

clean:
	rm -rf ./build

format:
	go install golang.org/x/tools/cmd/goimports@latest
	goimports -w=true ./
	gofmt -s=true -w=true ./
