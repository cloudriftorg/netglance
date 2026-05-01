# Install netglance with Docker

The simplest deployment. Works on any Linux host that has Docker.

## Prerequisites

- Docker 20.10+ with Compose v2
- A LAN to scan (host networking is needed for ARP to work)

## Run

```sh
git clone https://github.com/netglance/netglance
cd netglance
docker compose up -d
```

The web UI is at `http://<host-ip>:8473`. First load runs a setup wizard to
create the admin password.

## Container details

- Image: built from the repo's `Dockerfile` (Alpine + arp-scan + Go binary).
- Network: `network_mode: host` (required so arp-scan sees real interfaces).
- Storage: a named volume `netglance_data` mounted at `/data` inside the
  container. The SQLite DB lives there and survives recreates.
- Port: 8473/tcp. Override via the `NETGLANCE_BIND` env var.

## Updating

```sh
git pull
docker compose pull
docker compose up -d --build
```

## Why not bridge networking?

ARP is L2 — Docker's bridge isolates you from the host's L2, so arp-scan inside
a bridged container sees only the Docker network. Use `network_mode: host`,
or run on a Linux host with a bridge in promiscuous mode.

## macOS / Windows caveat

Docker Desktop runs containers in a Linux VM, so `network_mode: host` doesn't
do what you'd expect — you only see the VM's internal network. Fine for trying
the UI, but for real LAN scanning use a Linux host (or one of the native
install methods).
