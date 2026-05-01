# Netglance

Self-hosted, mobile-first LAN inventory: scopre i dispositivi sulla tua rete, traccia online/offline, manda notifiche email. Single binary Go + SQLite + frontend React embedded — distribuito come immagine Docker singola.

> Ispirato a WatchYourLAN e NetAlertX, riprogettato per essere leggero, multi-VLAN-aware e bello da usare anche dal telefono (PWA installabile).

## Caratteristiche

- 🔍 Scan TCP probe + ARP enrichment via `/proc/net/arp`
- 🏷️ Tag VLAN per ogni device (configurabile da UI)
- 📈 Storico online/offline + eventi per host
- 📧 Notifiche email (SMTP plain / STARTTLS / SMTPS) — endpoint test integrato
- 🔌 Schema config per integrazione gateway/router (OPNsense)
- 📱 PWA mobile-first, installabile su iOS/Android
- 🔐 Login admin locale, sessione cookie HttpOnly, `Secure` proxy-aware
- 🐳 Single container distroless nonroot, ~30MB
- ⚙️ Zero env vars per la logica: tutto via wizard al primo avvio

## Quick start

```yaml
# compose.yml
services:
  netglance:
    image: ghcr.io/massimoschiavop/netglance:latest
    container_name: netglance
    restart: unless-stopped
    network_mode: host
    cap_add:
      - NET_RAW
      - NET_ADMIN
    security_opt:
      - no-new-privileges:true
    read_only: true
    tmpfs:
      - /tmp
    mem_limit: 256m
    cpus: 1.0
    volumes:
      - netglance_data:/data
    healthcheck:
      test: ["CMD", "/netglance", "healthcheck"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 10s

volumes:
  netglance_data:
```

```bash
docker compose up -d
```

Apri `http://<host>:8080` → wizard di setup (admin + reti + SMTP) → fatto.

> **Nota**: `network_mode: host` è richiesto solo per scoprire device sulla LAN reale. In dev locale (Docker Desktop) puoi usare `ports: ["8080:8080"]` invece, ma vedrai solo i device della bridge Docker.

## Reverse proxy (Caddy / Traefik / nginx)

Netglance riconosce `X-Forwarded-Proto: https` e imposta correttamente il flag `Secure` sui cookie di sessione. Esempio Caddy:

```caddyfile
netglance.example.com {
    reverse_proxy 192.168.1.21:8080
}
```

## Sviluppo locale

Richiede Go 1.23+ e Node 22+.

```bash
make dev      # backend :8080 + frontend Vite :5173 (proxy /api)
make build    # produce ./netglance (Linux/amd64 di default)
make docker   # docker build -t netglance:dev .
make test     # go test ./...
```

## Documentazione

- [docs/PLAN.md](docs/PLAN.md) — piano originale di progettazione (architettura, schema DB, API, decisioni di design)
- [CHANGELOG.md](CHANGELOG.md) — release notes

## Licenza

MIT — vedi [LICENSE](LICENSE).
