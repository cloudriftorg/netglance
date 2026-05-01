# Piano — Netglance (progetto open-source generico)

WebApp self-hostable stile WatchYourLAN/NetAlertX, **mobile-first** + **PWA**, distribuita come **immagine Docker singola** (binario Go + frontend embedded). Inventario host LAN multi-VLAN, storico online/offline, notifiche email.

**Generica e distribuibile** — non specifica per Cloudrift. Pubblicabile su GitHub (licenza MIT/Apache-2.0) e GHCR. Chiunque la può adottare: niente subnet, gateway, SMTP, dominio o integrazioni hard-coded. Tutto configurabile dalla UI al primo avvio. I riferimenti a Cloudrift in questo piano (192.168.1.x, VLAN 1/20/30, `cloudrift.org`, OPNsense) sono **solo lo scenario di deploy di Massimo**, non default dell'app.

---

## Context

Cloudrift attualmente documenta NetAlertX come tool di inventario (vedi commit recenti). NetAlertX e WatchYourLAN sono validi ma:

- WatchYourLAN ha UI desktop-oriented, scarsa esperienza mobile
- NetAlertX è feature-rich ma pesante (PHP+Python, UI datata)
- Nessuno dei due gestisce in modo nativo l'**etichettatura per VLAN** del nostro setup (1/20/30)

L'obiettivo è un'app **leggera, mobile-first, integrata** con l'infrastruttura Cloudrift (SMTP relay OPNsense, deploy via Dockhand, reverse proxy Caddy, multi-VLAN trunk già attivo sulla VM `cloudrift-infrastructure`). Single binary Go + SQLite, niente DB esterni, niente runtime PHP.

---

## Decisioni utente

| Punto | Scelta |
|---|---|
| Repo | Nuovo repo separato — `netglance` (nome prodotto, distribuibile open-source) |
| Backend | Go + SQLite (`modernc.org/sqlite`, pure-Go, no CGO) |
| Frontend | React + Vite + TypeScript + Tailwind + shadcn/ui — PWA |
| Auth | Login locale (admin singolo, bcrypt, sessione cookie HttpOnly) |
| Feature V1 | Scan ARP/ICMP + inventario + MAC vendor + storico online/offline + grafici uptime + notifiche email + multi-VLAN + API REST + PWA |
| Scan multi-VLAN | Container `network_mode: host` su `cloudrift-infrastructure`, etichetta VLAN per ogni device |
| Deploy | `cloudrift-infrastructure` via Dockhand, dietro Caddy su `netglance.cloudrift.org` |

---

## Architettura

```
┌──────────── netglance container (network_mode: host) ────────────┐
│                                                                 │
│  ┌─ Scanner goroutine ──────────────────────────────────────┐   │
│  │  • ARP scan VLAN 1 (gopacket raw socket)                 │   │
│  │  • ICMP ping sweep VLAN 20/30 (routed via OPNsense)      │   │
│  │  • OPNsense API client → /api/diagnostics/interface/    │   │
│  │    getArp + dhcpv4/leases (MAC reali per VLAN 20/30)     │   │
│  │  • OUI lookup locale (IEEE oui.txt embedded)             │   │
│  └──────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              ▼                                  │
│  ┌─ Store (SQLite, /data/netglance.db) ──────────────────────┐   │
│  │  hosts, host_events, scans, settings, users              │   │
│  └──────────────────────────────────────────────────────────┘   │
│              │                              │                   │
│              ▼                              ▼                   │
│  ┌─ HTTP server (chi, :8080) ──┐  ┌─ Notifier ──────────────┐   │
│  │  /api/* (REST, JSON)        │  │  SMTP 192.168.1.1:25     │   │
│  │  /  → frontend embed (PWA)  │  │  no-auth, from           │   │
│  │  /healthz                   │  │  netglance@cloudrift.org  │   │
│  └─────────────────────────────┘  └──────────────────────────┘   │
└──────────────────────────────────────────────────────────────────┘
                              │
                              ▼
                   Caddy (OPNsense, *.cloudrift.org)
                   netglance.cloudrift.org → 192.168.1.21:8080
```

---

## Struttura repo `netglance`

```
netglance/
├── backend/
│   ├── cmd/server/main.go           # entrypoint, wiring
│   ├── internal/
│   │   ├── api/                     # router chi, handlers, middleware auth
│   │   ├── auth/                    # bcrypt, session cookie, /login /logout
│   │   ├── scanner/                 # arp.go, icmp.go, opnsense.go, scheduler.go
│   │   ├── store/                   # sqlite.go, migrations/, queries
│   │   ├── notify/                  # smtp.go (net/smtp, no auth)
│   │   ├── ouidb/                   # oui.txt embedded + lookup
│   │   ├── config/                  # env vars, config struct
│   │   └── webui/                   # embed.FS del frontend dist
│   ├── go.mod
│   └── go.sum
├── frontend/
│   ├── src/
│   │   ├── pages/                   # Hosts, HostDetail, Scans, Settings, Login
│   │   ├── components/              # HostCard, VlanBadge, UptimeChart, …
│   │   ├── lib/api.ts               # fetch wrapper
│   │   ├── pwa/                     # manifest, service worker
│   │   └── main.tsx
│   ├── tailwind.config.ts
│   ├── vite.config.ts               # PWA plugin (vite-plugin-pwa)
│   └── package.json
├── Dockerfile                       # multi-stage, distroless
├── compose.yml                      # esempio deploy Dockhand
├── Makefile
├── README.md
└── .github/workflows/build.yml      # build + push GHCR
```

---

## Schema dati (SQLite)

```sql
CREATE TABLE users (
  id INTEGER PRIMARY KEY, username TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL, created_at INTEGER NOT NULL
);

CREATE TABLE hosts (
  id INTEGER PRIMARY KEY,
  mac TEXT UNIQUE NOT NULL,           -- chiave logica
  ip TEXT NOT NULL,                   -- ultimo IP visto
  vlan_id INTEGER NOT NULL,           -- 1/20/30 (derivato da CIDR)
  hostname TEXT, vendor TEXT,
  custom_name TEXT,                   -- override utente
  first_seen INTEGER NOT NULL,
  last_seen INTEGER NOT NULL,
  online INTEGER NOT NULL DEFAULT 0,
  notify_offline INTEGER DEFAULT 1
);
CREATE INDEX idx_hosts_vlan ON hosts(vlan_id);
CREATE INDEX idx_hosts_online ON hosts(online);

CREATE TABLE host_events (
  id INTEGER PRIMARY KEY,
  host_id INTEGER NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
  ts INTEGER NOT NULL,
  kind TEXT NOT NULL,                 -- 'online'|'offline'|'new'|'ip_change'
  ip TEXT
);
CREATE INDEX idx_events_host_ts ON host_events(host_id, ts DESC);

CREATE TABLE scans (
  id INTEGER PRIMARY KEY, started_at INTEGER, ended_at INTEGER,
  vlan_id INTEGER, hosts_found INTEGER, error TEXT
);

CREATE TABLE settings (k TEXT PRIMARY KEY, v TEXT NOT NULL);
```

L'**uptime per host** si calcola al volo dagli `host_events` (coppie `online`/`offline`) — niente tabella di slot temporali, semplice e flessibile.

---

## API REST (autenticata via cookie sessione)

| Metodo | Path | Scopo |
|---|---|---|
| `POST` | `/api/login` | username+password → set-cookie sessione |
| `POST` | `/api/logout` | invalida sessione |
| `GET` | `/api/hosts?vlan=&online=&q=` | lista host con filtri |
| `GET` | `/api/hosts/:mac` | dettaglio + eventi recenti |
| `PATCH` | `/api/hosts/:mac` | `custom_name`, `notify_offline` |
| `GET` | `/api/hosts/:mac/uptime?range=24h\|7d\|30d` | serie eventi per grafico |
| `POST` | `/api/scan/run` | trigger scan manuale |
| `GET` | `/api/scans?limit=` | storico scan |
| `GET` | `/api/settings` / `PUT` | tutta la config runtime (reti, scan, SMTP, OPNsense, notifiche) |
| `POST` | `/api/settings/test-smtp` | invia email di prova al destinatario configurato |
| `POST` | `/api/settings/test-opnsense` | verifica credenziali API OPNsense |
| `GET` | `/api/setup/status` / `POST /api/setup` | first-run wizard (no-auth finché non c'è admin) |
| `GET` | `/healthz` | liveness (no auth) |

---

## Scanner — strategia multi-VLAN

1. **ARP nativo VLAN 1** via `gopacket` raw socket sull'interfaccia primaria della VM (richiede `cap_net_raw` → `Dockerfile` con `setcap`, container non-root).
2. **ICMP echo sweep** sui CIDR di VLAN 20 (`192.168.20.0/24`) e VLAN 30 (`192.168.30.0/24`) — routed via OPNsense. Concorrenza limitata (worker pool 32).
3. **OPNsense API client** (config opzionale, key+secret) per arricchire i risultati delle altre VLAN con MAC reali letti dalla ARP table del gateway:
   - `GET /api/diagnostics/interface/getArp` → mappa IP→MAC per tutte le VLAN
   - `GET /api/dhcpv4/leases/searchLease` (Dnsmasq) → hostname suggeriti
4. **Tag VLAN** assegnato in base alla CIDR di appartenenza dell'IP (configurabile in `/api/settings`).
5. **Scheduler** con interval configurabile (default 5 min) + run manuale via API.
6. **Stato online/offline**: un host risulta `offline` se non visto per N scan consecutivi (default 3) → genera evento + notifica email se abilitata.

---

## Frontend — UI mobile-first

Pagine core:

- **Hosts** (`/`): lista cards con filtro VLAN (chip in alto), toggle online/offline, search MAC/IP/nome, sticky header. Pull-to-refresh.
- **Host detail** (`/h/:mac`): info, grafico uptime 24h/7d/30d (Recharts), timeline eventi, edit `custom_name` e flag notifiche.
- **Scans** (`/scans`): cronologia scan, durata, host trovati, errori.
- **Settings** (`/settings`): unica pagina con tutta la configurazione runtime — niente env var nel compose, **tutto qui**:
  - **Reti**: lista subnet/VLAN modificabile (CIDR + nome + ID VLAN, opzionale). Al primo avvio il wizard **auto-rileva la subnet primaria** dell'interfaccia del container e la propone come unica voce iniziale; l'utente aggiunge le altre se ne ha. Nessuna subnet hard-coded.
  - **Scan**: intervallo (default 5 min), N scan mancati per dichiarare offline (default 3), interfaccia per ARP (auto-detect, override manuale).
  - **Gateway/Router integration** (opzionale, pluggable): adapter selezionabile fra `none` / `opnsense` / `pfsense` / `mikrotik` (V1: solo `none` + `opnsense`, gli altri come stub estendibile). Campi: URL, credenziali, verify TLS, pulsante **Test connessione**. Serve solo se vuoi MAC reali per subnet routed (no LAN diretta).
  - **SMTP**: host:porta, mittente, destinatari (lista), TLS on/off, auth on/off (user/pass se on), pulsante **Send test email**. **Tutti i campi vuoti di default** — l'utente li compila secondo la propria infrastruttura.
  - **Notifiche**: toggle per evento (`new`, `offline`, `back online`).
  - **Account**: cambio password admin.
- **Login** (`/login`): form unico. Al **primo avvio** redirect a `/setup` (wizard 3 step: crea admin → reti → SMTP) — nessuna configurazione richiesta da CLI o env.

PWA via `vite-plugin-pwa`: manifest, service worker offline-shell, icone, installabile da Safari/Chrome mobile.

Componenti shadcn/ui: `Card`, `Badge`, `Input`, `Sheet` (per filtri mobile), `Dialog`, `Tabs`. Tailwind con breakpoint `sm/md/lg` ma layout **mobile-first** (cards verticali, tabella `md:` solo sopra 768px).

---

## Notifiche email

`internal/notify/smtp.go` — client SMTP generico (`net/smtp` + `crypto/tls`) che supporta:

- plain (no auth, no TLS) — es. relay interno LAN-trusted
- STARTTLS + LOGIN/PLAIN — gmail, mailgun, ecc.
- SMTPS (TLS implicito su 465)

Tutti i parametri (host, port, from, to[], auth, TLS) presi dal DB `settings`, modificabili dalla pagina Settings. Eventi notificati (toggle per tipo):

- Nuovo MAC mai visto prima (kind=`new`)
- Host con `notify_offline=1` passa offline
- Host torna online dopo offline

---

## Dockerfile (multi-stage)

```dockerfile
# Stage 1: frontend build
FROM node:22-alpine AS fe
WORKDIR /app
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: backend build
FROM golang:1.23-alpine AS be
WORKDIR /src
COPY backend/go.* ./
RUN go mod download
COPY backend/ ./
COPY --from=fe /app/dist ./internal/webui/dist
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /netglance ./cmd/server

# Stage 3: runtime
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=be /netglance /netglance
EXPOSE 8080
USER nonroot
ENTRYPOINT ["/netglance"]
```

> ARP scan richiede raw socket → in produzione il container gira con `cap_add: [NET_RAW, NET_ADMIN]` e `network_mode: host`, non come root.

---

## compose.yml (esempio generico — qui il deploy specifico Cloudrift)

**Config minimale**: nessuna env var per logica applicativa. Tutto si configura dalla pagina **Settings** della UI dopo il primo accesso. Solo volume per persistenza DB.

```yaml
services:
  netglance:
    image: ghcr.io/<owner>/netglance:latest
    container_name: netglance
    restart: unless-stopped
    network_mode: host
    cap_add: [NET_RAW, NET_ADMIN]
    volumes:
      - netglance_data:/data
volumes:
  netglance_data:
```

> Stesso compose vale **per qualunque utente**, qualunque rete. Apri `http://<host>:8080`, esegui il wizard, fatto. Nessuna riga da modificare per il caso Cloudrift vs un altro homelab.

> Bind port `:8080` e data dir `/data` sono i default del binario. Se uno dei due deve cambiare, è sovrascrivibile con `NETGLANCE_BIND` / `NETGLANCE_DATA_DIR` ma di norma **non serve toccare nulla**.

Esposizione esterna via Caddy (pattern già documentato in `6-caddy-reverse-proxy-setup.md`):

| Caddy → Subdomains | `netglance.cloudrift.org` |
| Caddy → Handlers | Upstream `192.168.1.21:8080` |

---

## File critici da creare

| File | Ruolo |
|---|---|
| `backend/cmd/server/main.go` | entrypoint |
| `backend/internal/scanner/arp.go` | ARP via gopacket VLAN 1 |
| `backend/internal/scanner/icmp.go` | ping sweep VLAN 20/30 |
| `backend/internal/scanner/opnsense.go` | client API ARP/leases |
| `backend/internal/scanner/scheduler.go` | loop periodico + dedup MAC |
| `backend/internal/store/sqlite.go` + `migrations/0001_init.sql` | schema iniziale |
| `backend/internal/api/router.go` + handlers | REST endpoints |
| `backend/internal/auth/session.go` | login/cookie/middleware |
| `backend/internal/notify/smtp.go` | email su eventi |
| `backend/internal/ouidb/oui.go` + `oui.txt` | vendor lookup embedded |
| `frontend/src/pages/Hosts.tsx`, `HostDetail.tsx`, `Settings.tsx`, `Login.tsx` | UI principale |
| `frontend/src/pwa/manifest.webmanifest` + icone | PWA installabile |
| `Dockerfile`, `compose.yml`, `Makefile`, `README.md` | build & deploy |
| `.github/workflows/build.yml` | CI multi-arch (amd64+arm64) → GHCR |

---

## Verifica end-to-end

1. **Dev locale (host = Mac):**
   - `make dev` lancia backend `:8080` e Vite `:5173` (proxy `/api`).
   - Primo accesso → wizard `/setup`: crea admin, conferma reti, imposta SMTP. Nessuna CLI, nessuna env var.
   - Trigger scan manuale dalla UI → la VLAN 1 mostra il Mac stesso, IP 192.168.1.50.

2. **Build immagine:**
   - `docker build -t netglance:test .` → immagine < 30MB.
   - `docker run --rm -p 8080:8080 netglance:test` → UI accessibile.

3. **Deploy su `cloudrift-infrastructure` via Dockhand:**
   - Adopt stack da `/opt/netglance/compose.yml` (pattern guida 8). Compose minimale: solo `image`, `network_mode: host`, `cap_add`, `volumes`.
   - Apri `http://192.168.1.21:8080` → wizard di setup → completi reti, SMTP, OPNsense API in 30 secondi.
   - Verifica scan VLAN 1: tutti i device noti (OPNsense .1, TP-Link .2, AP .3/.4, Proxmox .20, Mac .50, PC .51) compaiono entro 5 min con vendor MAC corretto.
   - Verifica VLAN 20/30: configurare un device IoT noto, attendere scan, verificare badge VLAN corretto. Se l'integrazione OPNsense API è abilitata il MAC appare reale, altrimenti l'IP risulta visto ma senza MAC (limite atteso del routed-ping).

4. **Notifiche email:**
   - Spegnere device in lista watchlist → dopo 3 scan mancati arriva email da `netglance@cloudrift.org` al destinatario impostato.
   - Test manuale: pulsante "Send test email" in Settings.

5. **Caddy + PWA:**
   - `https://netglance.cloudrift.org` accessibile con cert wildcard.
   - Da Safari iOS → "Aggiungi alla schermata Home" → app si installa, icona corretta, splash, login funzionante offline shell.

6. **Test backend:**
   - `go test ./...` copre store, scanner parsing, OUI lookup, notify formatting.
   - Lint frontend: `npm run lint` + `tsc --noEmit`.

---

## Iterazioni successive (fuori scope V1)

- WebSocket per aggiornamenti host live (oggi: polling `/api/hosts` ogni 15s).
- Push notification web (richiede HTTPS pubblico + VAPID).
- Multi-utente con ruoli.
- Backup automatico DB su volume esterno.
- Sub-interfaces VLAN nel Debian host (`eth0.20`, `eth0.30`) per ARP nativo anche su IoT/Guests, eliminando dipendenza dall'API OPNsense.
