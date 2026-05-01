.PHONY: dev backend-dev frontend-dev build docker test tidy clean

dev:
	@echo "Backend: http://localhost:8080  |  Frontend (Vite): http://localhost:5173"
	@( cd backend && go run ./cmd/server ) & \
	 ( cd frontend && npm run dev ) ; \
	 wait

backend-dev:
	cd backend && go run ./cmd/server

frontend-dev:
	cd frontend && npm run dev

# Build static frontend, embed into Go binary, produce ./netglance.
# Linux/amd64 by default; override with GOOS/GOARCH if needed.
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
