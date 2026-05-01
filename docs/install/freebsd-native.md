# Install netglance on FreeBSD (native, no OPNsense)

For a vanilla FreeBSD box where you want netglance running as a regular
service without the OPNsense plugin layer.

> If you're on **OPNsense**, use the [plugin install](opnsense-plugin.md)
> instead — it gives you a configuration tab in the OPNsense UI.

## Prerequisites

- FreeBSD 14 (other releases may work, untested)
- Add the netglance pkg repo (one-time): see [opnsense-plugin.md](opnsense-plugin.md#1-add-the-pkg-repo) — same step.

## Install

```sh
pkg update
pkg install netglance
```

This pulls in `arp-scan` automatically. The pkg creates:
- `/usr/local/sbin/netglance` — the daemon binary
- `/usr/local/etc/rc.d/netglance` — the rc.d script
- `/var/db/netglance/` — state directory (owned by the `netglance` user)
- `/usr/local/etc/netglance/` — for an optional `netglance.env` file

## Configure

Create `/usr/local/etc/netglance/netglance.env` if you want to override
defaults (otherwise the daemon listens on `0.0.0.0:8473` and waits for
configuration via the web UI):

```sh
NETGLANCE_BIND=:8473
NETGLANCE_DATA_DIR=/var/db/netglance
# Optional: pre-seed the scan settings (then reset them via the UI later)
# NETGLANCE_SCAN_IFACES=igb0
# NETGLANCE_NETWORKS=192.168.1.0/24:0:lan
```

Enable + start:

```sh
sysrc netglance_enable=YES
service netglance start
service netglance status
```

UI: `http://<host-ip>:8473`. First load is the setup wizard.

## Logs

```sh
tail -F /var/log/netglance.log
```

## Uninstall

```sh
service netglance stop
pkg delete netglance
rm -rf /var/db/netglance     # wipes the database — irreversible
```
