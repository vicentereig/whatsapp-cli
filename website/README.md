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

This site is meant to be published at `oss.vicente.services/whatsapp-cli`,
alongside the other project pages hosted from the `vicente.services/website`
repo (see `sites/oss/` there — that's how `dspy.rb`'s docs are published
under `oss.vicente.services/dspy.rb`).

**The exact publish step from this repo into that one is not wired up yet.**
The known-working pattern, based on how `dspy.rb` publishes today:

1. Build or copy this directory's output into the `website` repo, under a
   path that maps to `oss.vicente.services/whatsapp-cli` (mirroring the
   `dist/oss.vicente.services/dspy.rb/` layout observed there).
2. Deploy the `website` repo as usual (it runs on Fly.io — see that repo's
   `fly.toml` and `README.md`).

Until this is confirmed and scripted, treat step 1 as **manual**: copy this
directory's contents into place in the `website` repo, verify the page
renders under `/whatsapp-cli`, then deploy that repo. If you formalize this
(a script, a CI job, a git submodule), update this section — the workflow
described here is what worked at the time of writing, not a guarantee of
how the pipeline works long-term.
