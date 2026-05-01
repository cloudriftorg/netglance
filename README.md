# Netglance

> ⚠️ **Work in progress**
>
> Project under active development. **Use at your own risk** — there is no
> serious test coverage and the API/data model can change in
> backwards-incompatible ways from one commit to the next. Feel free to
> clone, poke around and send PRs, but don't expect stability guarantees.

Self-hosted LAN inventory: discovers devices on the network via
`arp-scan`, tracks online/offline transitions and sends email
notifications. Single Go binary + SQLite + embedded React frontend,
shipped as one Docker image.

Inspired by [WatchYourLAN](https://github.com/aceberg/WatchYourLAN) and
[NetAlertX](https://github.com/jokob-sk/NetAlertX), redesigned to be
lightweight, multi-VLAN-aware and usable from a phone (installable PWA).

## Features

- 🔍 **ARP-based scanning** (`arp-scan`) with a modern UI and a multi-VLAN data model
- 🏷️ Per-device VLAN tags, configurable from the UI
- 📈 Online/offline history and per-host events
- 📧 Email notifications (SMTP plain / STARTTLS / SMTPS) with a test endpoint
- ⏱️ Configurable auto-scan (interval + on/off toggle), manual trigger from the UI
- 📱 Mobile-first PWA, installable on iOS/Android
- 🌓 Light / dark / system theme, persisted across reloads
- 🔐 Local admin login, HttpOnly session cookie, proxy-aware `Secure` flag
- 🐳 Single Alpine container, ~30 MB
- ⚙️ No env vars for app logic — everything is configured via a first-run wizard

## Quick start

`network_mode: host` is required: `arp-scan` sends raw ARP requests and
needs the host's real network stack, not a Docker bridge.

```yaml
# compose.yml
services:
  netglance:
    image: ghcr.io/massimoschiavop/netglance:latest
    restart: unless-stopped
    network_mode: host
    volumes:
      - netglance_data:/data

volumes:
  netglance_data:
```

```bash
docker compose up -d
```

Open `http://<host>:8080` → setup wizard (admin + networks + SMTP) → done.

### Multi-VLAN scanning

`arp-scan` is strictly L2: it only sees hosts in the broadcast domain of
the interface it runs on. To scan additional VLANs, the Docker host must
have a sub-interface (with an IP) in each VLAN you want to cover. Once
that is in place, add each CIDR + VLAN ID under Settings — the app
automatically picks the right interface for each network.

## Reverse proxy (Caddy / Traefik / nginx)

Netglance honors `X-Forwarded-Proto: https` and sets the `Secure` flag on
session cookies accordingly.

```caddyfile
netglance.example.com {
    reverse_proxy <netglance-host>:8080
}
```

## Local development

Only Docker is required.

```bash
make local         # build + run the whole app in Docker, http://localhost:8080
make logs          # tail logs
make local-stop    # stop
make reset         # wipe the local DB volume (next run = fresh setup)
```

> **macOS note**: the container only sees Docker Desktop's internal
> network, not the host's LAN. UI, settings, auth, migrations, vendor
> lookup and the scan loop are all testable. Real LAN/VLAN scanning
> requires a Linux host with `network_mode: host`.

### Frontend-only iteration

For frontend changes, a Vite dev server with HMR proxies `/api` to a
running backend:

```bash
make ui                                  # default backend (override via BACKEND=...)
make ui BACKEND=http://localhost:8080    # against a local netglance
```

### Other useful targets

```bash
make build     # static binary ./netglance (frontend embedded)
make docker    # build the netglance:dev image
make test      # go test ./...
make help      # full target list
```

## Documentation

- [CHANGELOG.md](CHANGELOG.md) — release notes

## License

MIT — see [LICENSE](LICENSE).
