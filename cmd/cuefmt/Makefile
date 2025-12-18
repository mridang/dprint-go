.PHONY: build clean lint

PLUGIN := cuefmt
OUT_DIR ?= build

build:
	mkdir -p $(OUT_DIR)
	# CUE is recursive; increase stack size to prevent "unreachable" (stack overflow) errors.
	tinygo build -o=$(OUT_DIR)/$(PLUGIN).wasm -target=wasm-unknown -scheduler=none -stack-size=4096kb -no-debug -opt=1 .
	go run ../addstart/main.go $(OUT_DIR)/$(PLUGIN).wasm $(OUT_DIR)/$(PLUGIN)-fixed.wasm
	mv $(OUT_DIR)/$(PLUGIN)-fixed.wasm $(OUT_DIR)/$(PLUGIN).wasm

clean:
	rm -rf $(OUT_DIR)

lint:
	golangci-lint run
