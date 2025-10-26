# Release Guide (v1.0.0 and onward)

This guide describes how to cut a tagged release that supports `go install`, pre-built binaries, and source builds for WhatsApp CLI.

## Semantic Versioning

- Use tags that follow `vMAJOR.MINOR.PATCH` (e.g., `v1.0.0`, `v1.0.1`).
- Stick to semver rules: increment PATCH for bug fixes, MINOR for backward-compatible features, MAJOR for breaking changes.
- `go install github.com/vicente/whatsapp-cli@v1.0.0` resolves to that tag automatically, so never retag an existing version.

## Pre-Release Checklist

1. Ensure the repo is clean and tests pass: `go test ./...`.
2. Run `go mod tidy` so dependency metadata matches the source.
3. Update documentation (README, QUICKSTART, etc.) with the new version and noteworthy changes.
4. Review the changelog/summary for the release notes.
5. Commit all changes before tagging.

## Tagging and Pushing

```bash
git tag -a v1.0.0 -m "v1.0.0"
git push origin v1.0.0
```

Replace `v1.0.0` with the appropriate version number. Pushing the tag triggers the release workflow.

## GitHub Release Automation

The workflow in `.github/workflows/release.yml` runs automatically for tags that match `v*`:

1. Executes `go test ./...` on Ubuntu to ensure the tag is healthy.
2. Builds CGO-enabled binaries for:
   - Linux: `amd64`, `arm64`
   - macOS: `amd64`, `arm64`
   - Windows: `amd64`
3. Packages binaries as:
   - Linux/macOS: `whatsapp-cli-<os>-<arch>.tar.gz`
   - Windows: `whatsapp-cli-windows-amd64.zip`
4. Generates SHA-256 checksum files for each archive and merges them into `checksums.txt`.
5. Publishes the artifacts and checksum bundle to the GitHub Release associated with the tag.

## Pre-Built Binary Distribution

After the workflow finishes:

1. Visit the GitHub Release page for the tag.
2. Verify that the tarballs/zip and `checksums.txt` are attached.
3. Add human-friendly release notes (highlights, breaking changes, upgrade instructions).
4. In documentation, reference the release download URLs.

Users should:

```bash
# Example for Linux amd64
wget https://github.com/vicentereig/whatsapp-cli/releases/download/v1.0.0/whatsapp-cli-linux-amd64.tar.gz
wget https://github.com/vicentereig/whatsapp-cli/releases/download/v1.0.0/checksums.txt
shasum -a 256 -c checksums.txt --ignore-missing
tar -xzf whatsapp-cli-linux-amd64.tar.gz
chmod +x whatsapp-cli-linux-amd64
sudo mv whatsapp-cli-linux-amd64 /usr/local/bin/whatsapp-cli
```

## Supporting `go install`

- Once the tag exists, `go install github.com/vicente/whatsapp-cli@v1.0.0` installs the release.
- If a patch is required, cut a new tag like `v1.0.1`. Never retag in place.

## Source Builds

The README already outlines the source-build flow:

```bash
git clone https://github.com/vicentereig/whatsapp-cli.git
cd whatsapp-cli
go mod download
go build -o whatsapp-cli .
```

Keep that section updated when dependencies or build flags change.

## Post-Release

1. Monitor issues or discussions for regressions.
2. Plan the next version number based on the nature of upcoming changes.
3. Keep `docs/RELEASE.md` updated if the process evolves (e.g., new targets or signing requirements).
