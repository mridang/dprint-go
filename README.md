# dprint-plugin-go

**dprint-plugin-go** is a formatter plugin for dprint that formats Go
files using Go’s canonical formatter. It’s compiled to WebAssembly with
TinyGo and plugs into dprint like any other plugin.

##### Why ?

Using a dprint plugin lets you keep a single, consistent formatting
workflow across polyglot repos. This plugin applies `go/format` so your
Go code follows the exact same rules as `gofmt`.

## Usage

Add the plugin to your **dprint** configuration and format your files.

```json
{
  "$schema": "https://dprint.dev/schemas/v0.json",
  "plugins": [
    "https://github.com/mridang/dprint-go/releases/download/v1.0.0/dprint.wasm"
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

### Options

This plugin mirrors `gofmt` and does not add custom options. If you pass
an override config from dprint, it is accepted but ignored.

## Caveats

None.

## Contributing

Contributions are welcome! If you find a bug or have suggestions for
improvement, please open an issue or submit a pull request.

## License

Apache License 2.0 © 2025 Mridang Agarwalla

