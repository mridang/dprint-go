.PHONY: build clean lint

PLUGIN := gofmt
OUT_DIR ?= build

build:
	mkdir -p $(OUT_DIR)
	tinygo build -o=$(OUT_DIR)/$(PLUGIN).wasm -target=wasm-unknown -scheduler=none -no-debug -opt=2 .
	go run ../addstart/main.go $(OUT_DIR)/$(PLUGIN).wasm $(OUT_DIR)/$(PLUGIN)-fixed.wasm
	mv $(OUT_DIR)/$(PLUGIN)-fixed.wasm $(OUT_DIR)/$(PLUGIN).wasm

clean:
	rm -rf $(OUT_DIR)

lint:
	golangci-lint run
