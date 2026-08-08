# Release Guide (v1.0.0 and onward)

This guide explains how to cut a tagged release. A release supports `go install`, pre-built binaries, and source builds for WhatsApp CLI.

## Semantic Versioning

- Use tags that follow `vMAJOR.MINOR.PATCH`, for example `v1.0.0` or `v1.0.1`.
- Follow semver rules: increment PATCH for bug fixes, MINOR for backward-compatible features, MAJOR for breaking changes.
- The command `go install github.com/vicentereig/whatsapp-cli@v1.0.0` resolves to that tag automatically. Never retag an existing version.

## Pre-Release Checklist

1. Make sure that the repo is clean and tests pass: `go test ./...`.
2. Run `go mod tidy` so dependency metadata matches the source.
3. Update the documentation (README, QUICKSTART, and other files) with the new version and notable changes.
4. Review the changelog for the release notes.
5. Commit all changes before tagging.

## Tagging and Pushing

```bash
git tag -a v1.0.0 -m "v1.0.0"
git push origin v1.0.0
```

Replace `v1.0.0` with the correct version number. When you push the tag, the release workflow starts.

## GitHub Release Automation

The workflow in `.github/workflows/release.yml` runs automatically for tags that match `v*`:

1. Runs `go test ./...` on Ubuntu to check that the tag is healthy.
2. Builds CGO-enabled binaries for:
   - Linux: `amd64`, `arm64`
   - macOS: `amd64`, `arm64`
   - Windows: `amd64`
3. Embeds the tag, for example `v1.0.0`, into the binary using Go ldflags. Then it smoke-tests each artifact by running `whatsapp-cli version`. It uses QEMU where native execution is not possible.
4. Packages binaries as:
   - Linux/macOS: `whatsapp-cli-<os>-<arch>.tar.gz`
   - Windows: `whatsapp-cli-windows-amd64.zip`
5. Generates a SHA-256 checksum file for each archive and merges the files into `checksums.txt`.
6. Signs every uploaded file — tarballs, zip files, and checksums — with Sigstore cosign, using GitHub's OIDC identity. This produces `.sig` and `.pem` companion files.
7. Publishes the artifacts, checksums, and cosign metadata to the GitHub Release for the tag.

## Pre-Built Binary Distribution

After the workflow finishes:

1. Visit the GitHub Release page for the tag.
2. Make sure that the tar files, the zip file, and `checksums.txt` are attached.
3. Add clear release notes (highlights, breaking changes, upgrade instructions).
4. In the documentation, link to the release download URLs.

To install the release, users run these commands:

```bash
# Example for Linux amd64
wget https://github.com/vicentereig/whatsapp-cli/releases/download/v1.0.0/whatsapp-cli-linux-amd64.tar.gz
wget https://github.com/vicentereig/whatsapp-cli/releases/download/v1.0.0/checksums.txt
shasum -a 256 -c checksums.txt --ignore-missing
tar -xzf whatsapp-cli-linux-amd64.tar.gz
chmod +x whatsapp-cli-linux-amd64
sudo mv whatsapp-cli-linux-amd64 /usr/local/bin/whatsapp-cli
```

To check provenance with Sigstore, run:

```bash
cosign verify-blob \
  --certificate whatsapp-cli-linux-amd64.tar.gz.pem \
  --signature whatsapp-cli-linux-amd64.tar.gz.sig \
  whatsapp-cli-linux-amd64.tar.gz
```

Expected identity:

- Issuer: `https://token.actions.githubusercontent.com`
- Subject/workflow: `repo:vicentereig/whatsapp-cli:ref:refs/tags/v*`

## Supporting `go install`

- Once the tag exists, `go install github.com/vicentereig/whatsapp-cli@v1.0.0` installs the release.
- If you need a patch, cut a new tag, for example `v1.0.1`. Never retag in place.

## Source Builds

The README already describes the source-build flow:

```bash
git clone https://github.com/vicentereig/whatsapp-cli.git
cd whatsapp-cli
go mod download
go build -o whatsapp-cli .
```

When dependencies or build flags change, keep that section updated.

## Post-Release

1. Monitor issues or discussions for regressions.
2. Plan the next version number to match the upcoming changes.
3. If the process changes, for example with new targets or signing requirements, keep `docs/RELEASE.md` updated.
