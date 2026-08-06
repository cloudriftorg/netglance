# Netglance

**Discover, inventory and watch every device on your LAN — from OPNsense.**

Netglance scans configured CIDRs over ARP (`arp-scan`), keeps a SQLite history of when each MAC was first/last seen and on which IP, and serves a mobile-friendly web UI on its own port (default `8473`). It installs as an OPNsense plugin: a `Services → Netglance` tab, wired into OPNsense's service lifecycle — start at boot, restart on save, status in the dashboard.

This is a **private plugin**, not published to the official OPNsense store. You build one self-contained installer, copy it to your firewall, and run it there. The daemon is a plain Go binary, so it also runs on any Linux or FreeBSD host — see [Native install](#native-install) below.

> ⚠️ Vibe-coded and not hardened. Don't put it on a production firewall you can't restore from a snapshot.

| Hosts page | Settings page | OPNsense plugin |
|---|---|---|
| ![Hosts page](docs/img/host_screen.png) | ![Settings page](docs/img/settings.png) | ![Services > Netglance](docs/img/opnsense_plugin.png) |

## Features

- 🔍 ARP-based discovery — fast, cheap, no agents
- 🏷️ Multi-VLAN data model — tag devices by VLAN, scan multiple subnets
- 📈 Online/offline history — per-host event timeline + uptime chart
- 🔔 Email notifications (SMTP plain / STARTTLS / SMTPS) on new device, offline, back-online
- ⏱️ Configurable auto-scan + manual trigger
- 📱 Mobile-first PWA, installable on iOS/Android
- 🌓 Light / dark / system theme
- 🔐 Local admin login, HttpOnly session cookie, proxy-aware

## Install

**1. Build** (needs `go` and `node`/`npm` here, nothing on the firewall):

```sh
./build.sh              # → dist/installer.sh  (~5 MB, freebsd/amd64)
./build.sh arm64        # for an ARM box
```

`dist/installer.sh` is committed, so a fresh clone already has a ready-to-run build and you only need `build.sh` when you change something. Rebuild before committing — and keep in mind each committed build adds ~6 MB to the repo forever, so commit the artifact when you're snapshotting a version that works, not on every experiment.

**2. Copy `dist/installer.sh` to the firewall** — scp, USB stick, GUI upload, whatever. `/tmp` is fine.

**3. Run it there as root:**

```sh
sh /tmp/installer.sh
```

It unpacks itself, installs `arp-scan` if missing, drops the files in place, reloads configd and php-fpm, cleans up. Nothing is downloaded or compiled on the firewall. Re-run the same file to update — `/var/db/netglance` is preserved.

Then hard-refresh the OPNsense GUI. Reloading configd and php-fpm picks up the plugin's menu entry, MVC classes and ACL most of the time, not always — if `Services → Netglance` doesn't show up, reboot the firewall.

`arp-scan` travels inside the installer: OPNsense's repo doesn't carry it, so `build.sh` pulls FreeBSD's package once (cached in `dist/deps/`) and the installer `pkg add`s it from disk. It pulls in nothing else — `libc` and `libpcap` are in the base system.

**4. Configure** in **Services → Netglance**: listen port, the interfaces to probe, then **Enable** and **Save**. The networks themselves (one row per CIDR, VLAN ID optional — it only drives the badge) are configured later, in netglance's own UI.

**Open Netglance UI** takes you to the web UI, which has its own admin password set on first launch — separate from your OPNsense login. Since the plugin isn't a `pkg`, it does not appear under System → Firmware → Plugins.

### What lands on the box

| Path | What |
|---|---|
| `/usr/local/sbin/netglance` | the daemon (static binary, UI embedded) |
| `/usr/local/bin/arp-scan` | from the bundled FreeBSD package, if not already there |
| `/usr/local/etc/rc.d/netglance` | rc script |
| `/usr/local/etc/inc/plugins.inc.d/netglance.inc` | service registration |
| `/usr/local/etc/rc.syshook.d/start/50-netglance` | start at boot |
| `/usr/local/opnsense/mvc/.../Netglance/` | GUI page, model, ACL, menu |
| `/usr/local/opnsense/service/conf/actions.d/actions_netglance.conf` | configd actions |
| `/usr/local/etc/netglance/netglance.env` | rendered from your settings on every Save |
| `/var/db/netglance/` | SQLite database |

No existing OPNsense file is modified. `/conf/config.xml` gains an `<OPNsense><netglance>` section, but only once you hit Save.

## Native install

Outside OPNsense there's no plugin, no GUI page and no config to inherit: the daemon runs on its own and everything is set from its web UI.

```sh
./build-native.sh                    # → dist/netglance-linux-amd64.tar.gz
./build-native.sh linux arm64
./build-native.sh freebsd amd64
```

Copy the tarball over, then on the host:

```sh
sudo apt install arp-scan            # pkg install arp-scan on FreeBSD
tar xzf netglance-linux-amd64.tar.gz && cd netglance-linux-amd64
sudo install -m 755 netglance /usr/local/bin/netglance
sudo install -m 644 netglance.service /etc/systemd/system/netglance.service
sudo mkdir -p /var/lib/netglance
sudo systemctl enable --now netglance
```

Open `http://<host>:8473`, set the admin password, and pick the interfaces to scan — that step is part of the first-run wizard here, since no orchestrator supplies them.

Two differences from the OPNsense path. `arp-scan` isn't bundled: a normal distro is one `apt`/`pkg` away from it, and pinning a version in the tarball would only get in the way. And VLAN tags are only recognised when the device name carries them (`eth0.20`, `enp1s0.30`) — nothing here knows your VLAN layout the way the firewall does. Set `NETGLANCE_IFACE_VLANS=iface=tag,…` in the unit file to fill that in.

The unit runs as root because arp-scan needs raw sockets; the file documents how to drop that with `CAP_NET_RAW` if you'd rather.

## Uninstall

```sh
sh /tmp/installer.sh uninstall
```

Stops the daemon and removes the binary, every plugin file, the state directory, the log, and the `<netglance>` section of `config.xml` — settings return to their pre-install state. `arp-scan` goes only if the installer was the one that pulled it in. OPNsense's own config history in `/conf/backup/` is left alone on purpose.

Any build works for this: the uninstall list is derived from the installer's own payload, so you can rebuild one instead of keeping the original.

## Repository layout

```
build.sh                     builds dist/installer.sh — the OPNsense plugin
build-native.sh              builds dist/netglance-<os>-<arch>.tar.gz — plain host
build-ui.sh                  builds the React UI into the Go embed dir; used by both
backend/                     Go daemon: API, ARP scanner, SQLite store, SMTP
frontend/                    React + Vite UI, embedded into the binary via //go:embed
deploy/opnsense-plugin/
  installer.sh               installer logic; build.sh appends the payload to it
  src/                       shipped verbatim to /usr/local on the firewall
deploy/native/               systemd unit for the non-OPNsense install
docs/img/                    screenshots
```

`src/` mirrors the target layout, so the install list and the uninstall list both come from it and can't drift apart.

## Development

```sh
cd frontend && npm run dev          # UI with HMR, proxies /api to $VITE_BACKEND_URL
cd backend && go test ./...
cd backend && go run ./cmd/server   # local daemon on :8473 (arp-scan needs root)
```

PHP/Volt/configd changes only run on OPNsense: rebuild, copy, re-run the installer — against a throwaway VM, not the firewall you depend on.

## Configuration

Listen port, scan interfaces and networks live in OPNsense's `config.xml` and are rendered into `/usr/local/etc/netglance/netglance.env` on every Save. Everything else (SMTP, notification toggles, per-host names/notes/watch flags, admin password) lives in netglance's own DB, configured from its web UI.

Behind a reverse proxy, netglance honors `X-Forwarded-Proto: https` and flips the session cookie's `Secure` flag accordingly.

## License

MIT — see [LICENSE](LICENSE).
