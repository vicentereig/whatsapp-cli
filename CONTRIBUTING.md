# Contributing

Thanks for your interest in whatsapp-cli. This project is maintained by one person in spare time. PRs are welcome and get reviewed. Feature requests without a PR wait until a contributor picks them up.

## Before you file an issue

Check the "Known issues" section below first. Your problem might already have a fix.

If it doesn't, include:

- The CLI version (`whatsapp-cli --version`).
- Your OS and architecture (e.g. macOS ARM64, Linux amd64).
- The exact command you ran.
- The full error output, including any log lines above the error.

Issue [#15](https://github.com/vicentereig/whatsapp-cli/issues/15) is a good model: it states the bug, gives exact repro steps, shows database evidence, and proposes a fix. Reports at that level get fixed fast.

## Known issues

### "Client outdated (405)" auth failures

This is the most common report ([#2](https://github.com/vicentereig/whatsapp-cli/issues/2), [#13](https://github.com/vicentereig/whatsapp-cli/issues/13), [#25](https://github.com/vicentereig/whatsapp-cli/issues/25)). WhatsApp periodically rotates the web client version it accepts. When that happens, the server sends `ConnectFailureClientOutdated (405)` and closes the connection.

The fix: bump the `whatsmeow` dependency and cut a new release. This recurs every few months — it is not a bug in this CLI. See the README's Connection Issues section for the exact steps. A PR that bumps `whatsmeow` and updates `go.sum` is always welcome when this happens; you don't need to ask first.

### "database is locked" during sync

The local store allows one writer at a time ([#19](https://github.com/vicentereig/whatsapp-cli/issues/19)). Don't run `sync` in two terminals at once, and don't run another store-writing command (like a history backfill) while `sync` is active. Doing so can corrupt the session, not just error out ([#20](https://github.com/vicentereig/whatsapp-cli/issues/20)).

### Auth succeeds but sync then fails or stalls

Run `sync` right after `auth` reports success. Waiting between the two steps has caused connection failures for some users ([#12](https://github.com/vicentereig/whatsapp-cli/issues/12)).

### `chats list` / `messages list` show only 20 results

That's the default `--limit`, not a bug ([#6](https://github.com/vicentereig/whatsapp-cli/issues/6), [#20](https://github.com/vicentereig/whatsapp-cli/issues/20)). Pass a higher `--limit` for scripts that expect the full store.

## Proposing a feature

1. Open an issue first. State the problem, then propose a design. Note any prior art (other projects that solved the same problem) if you know of any.
2. Wait for a green light on the design before you write the PR.
3. Send the PR. Mention any other open issue or PR your change might overlap with; the maintainer will coordinate a merge order or close the duplicate.

Issue [#20](https://github.com/vicentereig/whatsapp-cli/issues/20) is a model feature proposal: it states the problem, proposes a concrete CLI shape, cites prior art from other WhatsApp clients, links a working proof-of-concept branch, and lists caveats up front. Proposals at that level get accepted fast.

## Using AI coding agents

Agent-assisted PRs (Claude, Codex, or similar) are welcome. Point your agent at [AGENTS.md](./AGENTS.md) for build, test, and style conventions before it starts.

## What this project will and won't do for you

- PRs get reviewed and merged.
- Feature requests without a PR sit in the backlog until someone contributes one.
- Paid consulting is available if you want a feature built for you rather than contributing it yourself. Contact hey@vicente.services.

## Development setup

```bash
git clone https://github.com/vicentereig/whatsapp-cli.git
cd whatsapp-cli
go build
go test ./...
```

CI runs `go test ./...` on every push and pull request. Make sure it passes locally before you open a PR.
