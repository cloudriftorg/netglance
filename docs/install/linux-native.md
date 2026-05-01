# Install netglance on Linux (native, systemd)

For when you don't want a container — e.g. on a Raspberry Pi gateway, an LXC,
or a dedicated bare-metal Linux router.

## Prerequisites

- A modern Linux distro with systemd
- `arp-scan` (Debian/Ubuntu: `apt install arp-scan`; Fedora/RHEL: `dnf install arp-scan`)

## Install

```sh
# 1. Download the binary for your arch from the latest release
curl -L -o /usr/local/sbin/netglance \
  https://github.com/netglance/netglance/releases/latest/download/netglance-linux-amd64
chmod +x /usr/local/sbin/netglance

# 2. Create the unprivileged user
useradd --system --no-create-home --shell /usr/sbin/nologin netglance

# 3. Drop in the systemd unit
curl -L -o /etc/systemd/system/netglance.service \
  https://raw.githubusercontent.com/netglance/netglance/main/deploy/systemd/netglance.service

# 4. Start it
systemctl daemon-reload
systemctl enable --now netglance
systemctl status netglance
```

UI: `http://<host-ip>:8473`. State is at `/var/lib/netglance/netglance.db`.

## Customizing

```sh
systemctl edit netglance
```

Add a drop-in like:

```ini
[Service]
Environment=NETGLANCE_BIND=127.0.0.1:8473
```

Reload with `systemctl daemon-reload && systemctl restart netglance`.

## Logs

```sh
journalctl -u netglance -f
```

## Uninstall

```sh
systemctl disable --now netglance
rm /etc/systemd/system/netglance.service
rm /usr/local/sbin/netglance
userdel netglance
rm -rf /var/lib/netglance   # wipes the database — irreversible
```
