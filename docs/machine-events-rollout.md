# Machine login events — rollout & TODO

Status tracker for the `devlog login` machine-event feature. Survives a reboot:
pick up from the first unchecked box.

## What shipped

- **CLI** (`devlog-cli`, this repo): new `devlog login` command + `internal/machine`
  package. Sends entries with `type: "machine-event"` plus a structured
  `machine` object (eventKind, hostname, username, localIp, publicIp, osArch).
- **Backend** (`../devlog-daily-astro`): `entries.type` discriminator column
  (`log` | `machine-event`) + index, frontmatter `machine` metadata, API
  accepts/returns `type`+`machine`, web UI renders a 🖥️ badge (timeline +
  entry view).

## Remaining steps

- [ ] **Apply the DB migration to production.** In `../devlog-daily-astro`, run
      `pnpm db:push` against the prod database. This runs the Drizzle migrator
      over `src/db/migrations/`; migration `0002_outstanding_newton_destine.sql`
      adds the `type` column. It is idempotent (`ADD COLUMN IF NOT EXISTS`),
      safe to re-run. App startup does NOT auto-migrate — this is a manual/deploy
      step.
- [ ] **Deploy the backend** (`../devlog-daily-astro`) so the API and UI changes
      are live. Both repos' changes are already committed and pushed to `main`.
- [ ] **Verify end-to-end** once backend is deployed:
      `devlog login --dry-run` (offline preview), then a real
      `devlog login` against a configured API key, and confirm the entry shows
      the 🖥️ badge in the timeline at https://devlog.asd09.com.
- [ ] **Wire into machine login flows** (optional, per machine). Examples in
      `devlog login --help`:
      - macOS/Linux shell: add `devlog login >/dev/null 2>&1 &` to `~/.zprofile`.
      - Linux systemd user service: `ExecStart=/usr/local/bin/devlog login`.
      - Windows: startup task or PowerShell profile running `devlog login`.

## Release

- CLI release is cut with `make release` (default `BUMP=minor`): bumps the latest
  `v*` tag and pushes it; GitHub Actions + GoReleaser build and publish.
- `v0.7.0` is the first release containing `devlog login`.
