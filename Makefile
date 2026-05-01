.PHONY: help local local-stop logs reset ui build docker test tidy clean

help:
	@echo "netglance — make targets"
	@echo ""
	@echo "  make local        Build & run the full app in Docker (http://localhost:8473)"
	@echo "  make local-stop   Stop the local container"
	@echo "  make logs         Tail logs of the local container"
	@echo "  make reset        Wipe the local DB volume (next run = fresh setup)"
	@echo ""
	@echo "  make ui           Frontend dev server with HMR; proxies /api to a remote"
	@echo "                    backend (default: http://localhost:8473). Override:"
	@echo "                    make ui BACKEND=http://other:8473"
	@echo ""
	@echo "  make build        Static binary at ./netglance (frontend embedded)"
	@echo "  make docker       Build the Docker image as netglance:dev"
	@echo "  make test         Run Go tests"
	@echo "  make tidy         go mod tidy"
	@echo "  make clean        Remove build artifacts"

# ── Run the whole app locally in Docker ──────────────────────────────
# macOS caveat: the scanner sees only Docker's internal network (Docker
# Desktop runs containers in a Linux VM). Fine for UI / settings / DB
# migrations; for real LAN/VLAN scanning you need a Linux host with
# `network_mode: host` (see compose.yml).
local:
	docker compose -f compose.dev.yml up -d --build
	@echo "→ http://localhost:8473"

local-stop:
	docker compose -f compose.dev.yml down

logs:
	docker compose -f compose.dev.yml logs -f

reset:
	docker compose -f compose.dev.yml down -v

# ── Frontend HMR against a remote backend ────────────────────────────
BACKEND ?= http://localhost:8473
ui:
	@echo "→ http://localhost:5173  (proxying /api → $(BACKEND))"
	cd frontend && VITE_BACKEND_URL=$(BACKEND) npm run dev

# ── Build / utilities ────────────────────────────────────────────────
build:
	cd frontend && npm install --no-audit --no-fund && npm run build
	rm -rf backend/internal/webui/dist
	mkdir -p backend/internal/webui/dist
	cp -R frontend/dist/. backend/internal/webui/dist/
	cd backend && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ../netglance ./cmd/server

docker:
	docker build -t netglance:dev .

test:
	cd backend && go test ./...

tidy:
	cd backend && go mod tidy

clean:
	rm -rf netglance backend/internal/webui/dist frontend/dist
