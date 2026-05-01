.PHONY: help local local-stop logs reset ui build build-linux build-freebsd \
        release-freebsd docker test tidy clean \
        dev-vm-build dev-vm-deploy dev-plugin-sync dev-plugin-logs

help:
	@echo "netglance — make targets"
	@echo ""
	@echo "Local dev (Docker):"
	@echo "  make local             Build & run the full app in Docker (http://localhost:8473)"
	@echo "  make local-stop        Stop the local container"
	@echo "  make logs              Tail logs of the local container"
	@echo "  make reset             Wipe the local DB volume (next run = fresh setup)"
	@echo ""
	@echo "Frontend HMR:"
	@echo "  make ui                Frontend dev server (proxies /api → BACKEND, default localhost:8473)"
	@echo ""
	@echo "Builds:"
	@echo "  make build             Native binary ./netglance for the host OS (frontend embedded)"
	@echo "  make build-linux       Linux/amd64 binary at ./dist/netglance-linux-amd64"
	@echo "  make build-freebsd     FreeBSD/amd64 binary at ./dist/netglance-freebsd-amd64"
	@echo "  make release-freebsd   Tarball ./dist/netglance-freebsd-amd64.tar.gz (for the FreeBSD port)"
	@echo "  make docker            Build the Docker image as netglance:dev"
	@echo ""
	@echo "OPNsense plugin dev (requires VM_HOST set; e.g. make dev-plugin-sync VM_HOST=root@10.0.0.1):"
	@echo "  make dev-vm-build      Cross-build the binary for FreeBSD"
	@echo "  make dev-vm-deploy     scp the binary to VM:/usr/local/sbin/ and restart the service"
	@echo "  make dev-plugin-sync   rsync deploy/opnsense-plugin/src/ → VM and restart configd"
	@echo "  make dev-plugin-logs   tail configd + netglance logs on the VM"
	@echo ""
	@echo "Misc:"
	@echo "  make test              Run Go tests"
	@echo "  make tidy              go mod tidy"
	@echo "  make clean             Remove build artifacts"

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
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

# Embed the built React app into the Go binary. Shared by every build target.
.embed-frontend:
	cd frontend && npm install --no-audit --no-fund && npm run build
	rm -rf backend/internal/webui/dist
	mkdir -p backend/internal/webui/dist
	cp -R frontend/dist/. backend/internal/webui/dist/

build: .embed-frontend
	cd backend && CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o ../netglance ./cmd/server

build-linux: .embed-frontend
	mkdir -p dist
	cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build -trimpath -ldflags="$(LDFLAGS)" -o ../dist/netglance-linux-amd64 ./cmd/server

build-freebsd: .embed-frontend
	mkdir -p dist
	cd backend && CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 \
		go build -trimpath -ldflags="$(LDFLAGS)" -o ../dist/netglance-freebsd-amd64 ./cmd/server

# release-freebsd produces the source tarball that the FreeBSD port consumes
# (the port re-runs `go build` natively on the build host so symbols and CPU
# tuning match the target's pkg ABI; we only ship the source + bundled frontend).
release-freebsd: .embed-frontend
	mkdir -p dist
	tar --exclude='./dist' --exclude='./.git' --exclude='./frontend/node_modules' \
	    --exclude='./frontend/dist' --exclude='./.github' \
	    -czf dist/netglance-$(VERSION)-src.tar.gz \
	    ./backend ./frontend ./go.work* 2>/dev/null || \
	tar --exclude='./dist' --exclude='./.git' --exclude='./frontend/node_modules' \
	    --exclude='./frontend/dist' --exclude='./.github' \
	    -czf dist/netglance-$(VERSION)-src.tar.gz ./backend ./frontend

docker:
	docker build -t netglance:dev .

test:
	cd backend && go test ./...

tidy:
	cd backend && go mod tidy

clean:
	rm -rf netglance dist backend/internal/webui/dist frontend/dist

# ── OPNsense plugin dev loop (against a VM) ──────────────────────────
# VM_HOST must be passed in (e.g. "root@10.0.0.42") with SSH key auth set up.
# These targets exist so iterating on the plugin doesn't require a full pkg
# rebuild every change.
VM_HOST ?=

_check-vm:
	@if [ -z "$(VM_HOST)" ]; then \
		echo "ERROR: pass VM_HOST=root@<ip-of-opnsense-vm>" >&2; exit 1; \
	fi

dev-vm-build: build-freebsd

dev-vm-deploy: _check-vm dev-vm-build
	scp dist/netglance-freebsd-amd64 $(VM_HOST):/usr/local/sbin/netglance.new
	ssh $(VM_HOST) 'service netglance stop 2>/dev/null; \
		mv /usr/local/sbin/netglance.new /usr/local/sbin/netglance && \
		chmod +x /usr/local/sbin/netglance && \
		service netglance start || true'

dev-plugin-sync: _check-vm
	rsync -av --delete deploy/opnsense-plugin/src/ $(VM_HOST):/usr/local/opnsense/
	ssh $(VM_HOST) 'service configd restart && \
		configctl netglance reconfigure 2>/dev/null || true'

dev-plugin-logs: _check-vm
	ssh $(VM_HOST) 'tail -F /var/log/configd.log /var/log/netglance.log 2>/dev/null'
