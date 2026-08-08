# website/

The static marketing page for whatsapp-cli. Plain HTML, CSS, and vanilla JS —
no build step, no dependencies.

```
website/
├── index.html   # the page
├── tokens.css   # design tokens (colour, type, spacing)
├── style.css    # component + layout styles
├── demo.js      # tab switching, typing animation, copy-to-clipboard
└── README.md    # this file
```

## Preview locally

Any static file server works. From this directory:

```bash
python3 -m http.server 8000
# then open http://localhost:8000
```

Or just open `index.html` directly in a browser — nothing here depends on a
server (fonts load from Google Fonts over HTTPS; everything else is local).

## Editing

- **Copy and command reference** — keep it in sync with `../README.md`.
  If a flag or command changes there, update the table in `index.html`.
- **Terminal transcripts** (`demo.js`) — both are scripted replays, not a
  live WhatsApp connection: `SYNC_LINES` is whatsapp-cli's real documented
  `sync` progress output; `AGENT_LINES` is an illustrative sample session for
  a companion script, not a shipped `whatsapp-cli` subcommand. Keep both
  clearly labelled as illustrative if you change them — don't present sample
  data as a real backup.
- **Design tokens** — all colour and font values live in `tokens.css`. Don't
  add inline colours or fonts in `style.css` or `index.html`.

## Deploying to oss.vicente.services/whatsapp-cli

This site is published at `oss.vicente.services/whatsapp-cli` by the
`vicente.services/website` repo's `bin/assemble-sites` script, the same
mechanism that publishes `dspy.rb`'s docs under `oss.vicente.services/dspy.rb`.

That script expects this repo as a sibling checkout: `../whatsapp-cli` next
to `website/`. It copies `website/` (this directory, verbatim — no build
step) into `dist/oss.vicente.services/whatsapp-cli/`, then `website`'s
Dockerfile bakes `dist/` into the Fly image that serves `oss.vicente.services`.

To publish an update:

```bash
# from a checkout of vicente.services/website, with whatsapp-cli checked
# out as a sibling directory:
./bin/assemble-sites
./bin/verify-assembled-sites   # sanity checks before deploying
# then deploy the website repo as usual (Fly.io — see its fly.toml)
```

`bin/assemble-sites` records this repo's commit SHA (and the other sites')
in `dist/BUILD-METADATA.txt`, and can be pinned to a specific commit with
`WHATSAPP_CLI_SHA=<sha>` / pointed at a different checkout with
`WHATSAPP_CLI_SITE_DIR=<path>`.
