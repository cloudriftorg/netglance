# Changelog

All notable changes to this project will be documented here.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Removed
- Scans history page and `GET /api/scans` endpoint. The Hosts page already
  surfaces a "Last scan" badge sourced from a single record; an
  append-only scan log was unused dead weight.
- `scans` table replaced by a single `lastScan` JSON value in the
  `settings` table. No more zombie rows from crashed scans, no
  unbounded growth, no need for periodic pruning.
- `StartScan` / `FinishScan` / `LastFinishedScan` store API replaced by
  `RecordScan(LastScan)` / `GetLastScan()`.

### Fixed
- Auto-scan would silently stall once an arp-scan invocation hung — the
  global in-flight lock would never release. Two changes:
  - arp-scan now receives the explicit configured CIDR instead of the
    `-l` localnet flag, so it scans exactly the user's subnet (not the
    full /16 a Docker bridge advertises in dev).
  - Each arp-scan invocation runs under a 60s timeout context.
- Auto and manual scans now share a single in-flight flag (hoisted from
  `api/scan.go` into `scanner/inflight.go`). `/api/scan/status` reflects
  either source, so the Hosts-page spinner lights up during periodic
  scans too.
- The "Scan complete" toast no longer fires for periodic scans — only
  user-initiated scans surface a notification.

### Added
- `ScanEnabled` toggle in Settings (default on). Disables the periodic
  scanner; manual scans from the UI still work.
- `docs/multi-vlan-scanning.md` — VLAN sub-interface setup guide
  (Proxmox + Debian).
- `compose.dev.yml` + `make local` — single-command build & run for
  local development without Go/Node on the host.
- `make ui` — frontend dev server with HMR proxying `/api` to a remote
  backend (`BACKEND=…` to override target).

### Changed
- Scanner rewritten to use `arp-scan` (same methodology as WatchYourLAN)
  in place of TCP probe + `/proc/net/arp` enrichment. Real MACs come
  directly from ARP replies, vendors come from arp-scan's bundled OUI
  database (built-in OUI map kept as fallback).
- `OfflineAfter` default lowered from 3 to 1 — hosts go offline within
  one scan cycle, matching WatchYourLAN's behavior.
- `ScanEverySeconds` default lowered from 300 s to 120 s.
- Scan trigger button on the Hosts page is now a circular icon-only
  control. Spinner reflects any in-progress scan, manual or automatic.
- Runtime image switched from distroless to Alpine to bundle `arp-scan`.
- Setup wizard wording: removed reference to gateway integration.

### Removed (earlier in unreleased cycle)
- Gateway / OPNsense integration scaffolding (config struct, settings
  field, UI section). Was schema-only and never wired to a real client;
  multi-VLAN sub-interface support obsoletes the original motivation.
- `PrimaryIface` setting — superseded by per-network CIDR auto-detect.
- TCP probe code path (`probe.go`).

## [0.1.0] — 2026-05-01

First public release.

### Added
- Setup wizard (admin user, optional initial networks/SMTP)
- Local username/password auth with bcrypt + HttpOnly session cookies
- Cookie `Secure` flag is proxy-aware (`X-Forwarded-Proto: https`)
- Hosts list (table, full-width, responsive columns), filters by VLAN /
  online state / search; bulk auto-refresh polling
- Host detail with custom name, custom vendor, NEW flag, notify-offline
  toggle, recent events log
- Async manual scan with `/api/scan/status` polling and inline spinner
- Scans history page
- Settings page (networks, scan interval, SMTP, gateway adapter schema,
  notification toggles)
- Toast notifications (info/success/error, centered, click to dismiss)
- TCP probe sweep across configured CIDRs, MAC enrichment via
  `/proc/net/arp`
- SMTP send with plain / STARTTLS / implicit-TLS modes; test endpoint
- Built-in OUI vendor lookup for common homelab vendors
- PWA: manifest, service worker, installable, SVG icon
- `/healthz` checks DB connectivity
- Distroless nonroot Docker image (~30 MB) with `HEALTHCHECK`
- JSON structured logging
- `netglance healthcheck` subcommand for container health probe
- `netglance version` subcommand
- GitHub Actions: vet, test, lint, multi-arch build (amd64+arm64),
  publish to GHCR on tag and main branch

### Known limitations
- Notifier wiring: scan events are recorded but emails are not yet
  sent automatically (only manual SMTP test works)
- OPNsense API client not implemented; the gateway adapter section in
  Settings is schema-only
- No password reset flow — recovering requires DB reset

[0.1.0]: https://github.com/massimoschiavop/netglance/releases/tag/v0.1.0
