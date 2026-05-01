# Netglance

> ⚠️ **Work in progress · uso personale**
>
> Progetto in fase di sviluppo, scritto in **vibe coding** (pair-programming
> con un LLM) per la mia rete homelab. **Non pensato per la produzione di
> qualcun altro** — manca testing serio e l'API può cambiare in modo non
> retrocompatibile da un commit all'altro. Sentiti libero di clonarlo,
> guardarci dentro, mandare PR, ma non aspettarti garanzie di stabilità.

Self-hosted LAN inventory: scopre i dispositivi sulla rete via `arp-scan`,
traccia online/offline, manda notifiche email. Single binary Go + SQLite +
frontend React embedded, distribuito come singola immagine Docker.

> Ispirato a [WatchYourLAN](https://github.com/aceberg/WatchYourLAN) e
> [NetAlertX](https://github.com/jokob-sk/NetAlertX), riprogettato per
> essere leggero, multi-VLAN-aware e usabile dal telefono (PWA installabile).

## Caratteristiche

- 🔍 **Scan ARP-based** (`arp-scan`) — stessa metodologia di WatchYourLAN, con UI moderna e schema multi-VLAN
- 🏷️ Tag VLAN per device, configurabile da UI
- 📈 Storico online/offline + eventi per host
- 📧 Notifiche email (SMTP plain / STARTTLS / SMTPS) con endpoint test
- ⏱️ Auto-scan configurabile (intervallo + on/off toggle), trigger manuale dall'UI
- 📱 PWA mobile-first, installabile su iOS/Android
- 🔐 Login admin locale, sessione cookie HttpOnly, `Secure` proxy-aware
- 🐳 Single container Alpine, ~30 MB
- ⚙️ Zero env vars per la logica: tutto via wizard al primo avvio

## Quick start (produzione / homelab)

`network_mode: host` è obbligatorio: `arp-scan` invia ARP request raw e ha
bisogno della network stack reale dell'host, non di una bridge Docker.

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

Apri `http://<host>:8080` → wizard di setup (admin + reti + SMTP) → fatto.

### Scan multi-VLAN

`arp-scan` è strettamente L2: vede solo gli host nel broadcast domain
dell'interfaccia su cui gira. Per scansionare VLAN aggiuntive, l'host
Docker deve avere una sub-interface (con IP) in ogni VLAN che vuoi
coprire. Una volta su, basta aggiungere ogni CIDR + VLAN ID nelle
Settings di netglance — l'app trova in automatico l'interfaccia giusta
per ciascuna rete.

Setup passo-passo (Proxmox + Debian) in
[`docs/multi-vlan-scanning.md`](docs/multi-vlan-scanning.md).

## Reverse proxy (Caddy / Traefik / nginx)

Netglance riconosce `X-Forwarded-Proto: https` e imposta correttamente il
flag `Secure` sui cookie di sessione.

```caddyfile
netglance.example.com {
    reverse_proxy 192.168.1.21:8080
}
```

## Sviluppo locale

Serve solo Docker. Niente Go, niente Node sul Mac.

```bash
make local         # build + run dell'app intera in Docker, http://localhost:8080
make logs          # tail dei log
make local-stop    # spegne
make reset         # azzera il volume DB (next run = setup fresco)
```

> **macOS**: il container vede solo la rete interna di Docker Desktop, non
> la LAN del Mac (limite di Docker Desktop). UI, settings, auth, migrations,
> vendor lookup, scan loop → tutto testabile. Per scan reale su LAN/VLAN
> serve un host Linux con `network_mode: host`.

### Iterazione veloce sulla UI

Per cambi al solo frontend, dev server con HMR e proxy `/api` verso un
backend già attivo (default: VM in `192.168.1.21:8080`):

```bash
make ui                                  # default backend
make ui BACKEND=http://localhost:8080    # contro un netglance locale
```

Salvi un `.tsx`, la pagina si aggiorna in <1 s. Nessun rebuild.

### Altri target utili

```bash
make build     # binario statico ./netglance (frontend embedded)
make docker    # immagine netglance:dev
make test      # go test ./...
make help      # elenco completo
```

## Documentazione

- [docs/PLAN.md](docs/PLAN.md) — piano originale di progettazione
- [docs/multi-vlan-scanning.md](docs/multi-vlan-scanning.md) — setup VLAN sub-interfaces
- [CHANGELOG.md](CHANGELOG.md) — release notes

## Licenza

MIT — vedi [LICENSE](LICENSE).
