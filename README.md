# dprint-plugin-goat

**dprint-plugin-goat** provides two formatter plugins for dprint: one for Go files using Go's canonical formatter, and one for shell scripts using shfmt. Both are compiled to WebAssembly with TinyGo and plug into dprint like any other plugin.

##### Why ?

Using dprint plugins lets you keep a single, consistent formatting workflow across polyglot repos. The gofmt plugin applies `go/format` so your Go code follows the exact same rules as `gofmt`, while the shfmt plugin formats shell scripts with the same power as the standalone shfmt tool.

## Usage

### gofmt

Add the gofmt plugin to your **dprint** configuration to format Go files.

```json
{
  "$schema": "https://dprint.dev/schemas/v0.json",
  "plugins": [
    "https://github.com/mridang/dprint-goat/releases/download/v1.0.0/gofmt.wasm"
  ],
  "includes": [
    "**/*.go"
  ],
  "excludes": [
    "**/vendor"
  ]
}
```

```bash
dprint fmt --log-level=debug
```

#### Options

This plugin mirrors `gofmt` and does not add custom options. If you pass an override config from dprint, it is accepted but ignored.

### shfmt

Add the shfmt plugin to your **dprint** configuration to format shell scripts.

```json
{
  "$schema": "https://dprint.dev/schemas/v0.json",
  "plugins": [
    "https://github.com/mridang/dprint-goat/releases/download/v1.0.0/shfmt.wasm"
  ],
  "includes": [
    "**/*.sh"
  ]
}
```

```bash
dprint fmt --log-level=debug
```

#### Options

This plugin mirrors `shfmt` and does not add custom options. If you pass an override config from dprint, it is accepted but ignored.

## Caveats

None.

## Contributing

Contributions are welcome! If you find a bug or have suggestions for improvement, please open an issue or submit a pull request.

## License

Apache License 2.0 © 2025 Mridang Agarwalla
