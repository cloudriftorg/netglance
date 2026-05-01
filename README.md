# Netglance

Self-hosted, mobile-first LAN inventory: scopre i dispositivi sulla tua rete, traccia online/offline, manda notifiche email. Single binary Go + SQLite + frontend React embedded — distribuito come immagine Docker singola.

> Ispirato a WatchYourLAN e NetAlertX, riprogettato per essere leggero, multi-VLAN-aware e bello da usare anche dal telefono (PWA installabile).

## Caratteristiche

- 🔍 Scan ARP nativo + ICMP sweep multi-subnet
- 🏷️ Tag VLAN per ogni device (configurabile da UI)
- 📈 Storico online/offline + grafici uptime
- 📧 Notifiche email (SMTP plain / STARTTLS / SMTPS)
- 🔌 Adapter opzionale per gateway (OPNsense API per arricchire MAC delle subnet routed)
- 📱 PWA mobile-first, installabile su iOS/Android
- 🔐 Login admin locale, sessione cookie HttpOnly
- 🐳 Single container, `network_mode: host`, ~30MB
- ⚙️ Zero env vars per la logica: tutto via wizard al primo avvio

## Quick start

### macOS (binario nativo)

Docker Desktop su Mac non espone la rete reale al container, quindi per scansionare davvero la tua LAN usa il binario nativo:

```bash
make build-mac-arm64       # Apple Silicon — usa build-mac-amd64 per Intel
./netglance-darwin-arm64
```

Apri `http://localhost:8080` → wizard → in Settings aggiungi la tua subnet (es. `192.168.1.0/24`) → "Scan now". Il binario legge la ARP table di macOS via `arp -an` e arricchisce i probe TCP con i MAC reali.

### Linux (deploy production)

```yaml
services:
  netglance:
    image: netglance:dev
    container_name: netglance
    restart: unless-stopped
    network_mode: host
    cap_add: [NET_RAW, NET_ADMIN]
    volumes:
      - netglance_data:/data
volumes:
  netglance_data:
```

```bash
docker compose up -d
```

Apri `http://<host>:8080` → wizard di setup → fatto.

## Sviluppo

```bash
make dev        # backend :8080 + frontend Vite :5173 (proxy /api)
make docker     # build immagine docker locale
make test       # go test ./...
```

## Licenza

MIT — vedi [LICENSE](LICENSE).
