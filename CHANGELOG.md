# Changelog

All notable changes to this project will be documented in this file.

## v1.3.3 - 2026-04-11

### Fixed

- Upgraded `whatsmeow` to fix the `Client outdated (405)` authentication failure in newer WhatsApp sessions.
- Corrected subcommand flag parsing so commands like `messages search` and `contacts search` parse flags reliably.
- Improved archive checksum parsing during release validation.

### Improved

- Added dependency injection and more reliable CLI command wiring to reduce runtime failures.
- Added image sending support to `send`.
- Added standard CI on pushes and pull requests so test coverage runs outside release tags.

### Thanks

- `@seannetlife` for shipping the `whatsmeow` compatibility fix in PR #14.
- `@rmorgans` for the CLI robustness and image sending work in PR #9.
- `@spamsch` for the flag parsing fix in PR #10.
