# Changelog

All notable changes to this project will be documented here.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

### Added
- `ScanEnabled` toggle in Settings (default on). Disables the periodic
  scanner; manual scans from the UI still work.
- `docs/multi-vlan-scanning.md` — VLAN sub-interface setup guide
  (Proxmox + Debian).
- `compose.dev.yml` + `make local` — single-command build & run for
  local development without Go/Node on the host.
- `make ui` — frontend dev server with HMR proxying `/api` to a remote
  backend (`BACKEND=…` to override target).

### Removed
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
