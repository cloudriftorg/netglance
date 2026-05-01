# Changelog

All notable changes to this project will be documented here.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/);
this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
