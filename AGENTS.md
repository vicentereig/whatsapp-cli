# AGENTS.md

WhatsApp CLI is a single Go binary. It reads, searches, and sends WhatsApp
messages from the command line. It talks to WhatsApp through the `whatsmeow`
library, not through an official WhatsApp API.

## Build, run, test

```bash
go mod download                 # install dependencies
go build -o whatsapp-cli .      # build the binary
go test ./...                   # run all tests (the command CI runs)
go test -race ./...             # run tests with the race detector
go test -cover ./...            # run tests with coverage
go test ./internal/store        # run one package
```

If the sandbox cannot write to the global Go module cache, point
`GOMODCACHE` at a folder inside the repo:

```bash
GOMODCACHE=$(pwd)/.gomodcache go build ./...
```

To embed a version string at build time, use `-ldflags`:

```bash
go build -ldflags "-X main.version=X.Y.Z" -o whatsapp-cli .
```

Run the binary like this:

```bash
./whatsapp-cli auth
./whatsapp-cli sync
./whatsapp-cli --store /custom/path chats list
```

CI (`.github/workflows/ci.yml`) runs only `go test ./...`, on every push and
pull request. There is no linter config and no Makefile in this repo.

## Code style

- Package layout: `internal/client` wraps `whatsmeow`. `internal/store`
  wraps SQLite. `internal/commands` holds the `App` type, with one method
  per CLI command. `internal/output` formats the JSON output.
  `internal/types` holds types shared between `client` and `commands`, to
  avoid an import cycle.
- Define interfaces at the consumer, not at the implementation.
  `internal/commands/interfaces.go` defines the `WAClient` and
  `MessageStore` interfaces. The concrete types live in `internal/client`
  and `internal/store`.
- Give each component two constructors. `NewApp()` builds it with real
  dependencies. `NewAppWithDeps()` takes injected mocks, for tests.
- Wrap errors with context: `fmt.Errorf("failed to X: %v", err)`.
- Command methods on `App` (`Auth`, `Sync`, `ListMessages`, `SendMessage`,
  and so on) return a JSON string, not a Go `error`. Each method calls
  `output.Success(data)` or `output.Error(err)` at its boundary.
- Every CLI command must print this JSON shape to stdout:
  ```json
  {"success": true, "data": ..., "error": null}
  ```
- Go's `flag` package stops parsing at the first non-flag argument. For a
  subcommand like `messages list --chat X`, parse flags from `args[2:]`,
  skipping both the command and the subcommand name. See `main.go` and the
  regression tests in `main_test.go` (GitHub issue #5).

## Architecture

See `docs/architecture.md` for the full design. In short: `main.go` parses
flags and calls into `internal/commands.App`. `App` holds a `WAClient`
(talks to WhatsApp through `whatsmeow`) and a `MessageStore` (reads and
writes SQLite). Session data lives in `store/whatsapp.db`. Message history
lives in `store/messages.db`. Downloaded media lives under `store/media/`.

## Testing

- Tests use `testify/require`. There is no VCR setup and no mocking
  framework. Mocks are hand-written structs in `*_test.go` files, for
  example `internal/commands/mocks_test.go`, that implement the `WAClient`
  and `MessageStore` interfaces.
- Tests run against these mocks. They never open a real WhatsApp
  connection.
- Name regression tests after the bug they fix. `main_test.go` and
  `internal/commands/behavioral_test.go` name the GitHub issue or pull
  request in the test name and comments. Follow this pattern for new
  regression tests.
- `testdata/` exists but is empty. It is gitignored, so this repo does not
  currently ship fixture files.

## Constraints

- `whatsmeow` uses the unofficial WhatsApp Web protocol, not a
  WhatsApp-published API. A protocol change on WhatsApp's side can break
  the client without warning. Commit `bf602c5` fixed one such break (a
  "Client outdated (405)" auth error). Check `CHANGELOG.md` for the
  history.
- The session lives in `store/whatsapp.db`, created by scanning a QR code.
  It normally lasts about 20 days.
- `store/` holds the session and the full message history. Never commit
  it. `.gitignore` already excludes `/store/`.
- The `go-sqlite3` driver needs CGO. Cross-compiling for another platform
  needs `CGO_ENABLED=1` and, for Linux ARM64, a cross-compiler. See
  `.github/workflows/release.yml` for the exact build matrix.

## More docs

- `README.md` — full command reference, FAQ, troubleshooting
- `docs/architecture.md` — design and data flow
- `docs/RELEASE.md` — how to cut and publish a release
- `QUICKSTART.md` — five-minute setup
- `CHANGELOG.md` — release history
