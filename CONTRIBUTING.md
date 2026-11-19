# Contributing to dprint-go

Thank you for your interest in contributing!

## Before you start

- Open an issue first for non-trivial changes so the approach can be agreed on
  before you invest time writing code.
- For typo fixes, doc improvements, and small bug fixes a PR without an issue
  is fine.

## Development setup

```bash
# Install devbox (https://www.jetify.com/devbox/)
devbox shell          # activates Go 1.27, TinyGo, dprint, etc. (golangci-lint and gotestsum run through `go tool`)
make build            # build every plugin to build/*.wasm
make test             # run unit tests
make lint             # golangci-lint
make format           # dprint + gofmt
```

See the [README](README.md) for how to use the plugins with dprint.

## Adding a plugin

Every plugin is one directory under `cmd/`, built to one WebAssembly file.

1. Create `cmd/<name>/` with a `main.go` that registers the formatter with
   `internal/dprint`, a `stubs.go`, a `Makefile`, a `VERSION` file and the
   `dprint.json` used by its tests. Copy an existing plugin as a starting point.
2. Write `main_test.go`, `api_test.go` and `dprint_test.go` next to it. The tests
   load the compiled WebAssembly module, so a plugin is only done when it passes
   through the real module and not just through the Go function.
3. Add `build-<name>` and `test-<name>` targets to the root `Makefile`, add the
   plugin to the `build` target and to `.releaserc.json`.
4. Document the plugin and its options in the README.

## Pull request checklist

- [ ] `make test` passes
- [ ] `make lint` is clean
- [ ] `make format` produces no diff
- [ ] Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/)
      (`feat:`, `fix:`, `docs:`, `chore:` etc.) — semantic-release uses these to
      determine the next version

## Code style

- Standard `gofmt`/`goimports` formatting (enforced by CI)
- All exported symbols must have a doc comment
- The `VERSION` files are written by the release process — do not edit them by
  hand

## License

By contributing, you agree your contributions will be licensed under the
[Apache-2.0 License](LICENSE).
