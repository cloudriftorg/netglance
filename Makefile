.PHONY: dev backend-dev frontend-dev build docker test tidy clean build-mac build-mac-arm64 build-mac-amd64

dev:
	@echo "Backend: http://localhost:8080  |  Frontend (Vite): http://localhost:5173"
	@( cd backend && go run ./cmd/server ) & \
	 ( cd frontend && npm run dev ) ; \
	 wait

backend-dev:
	cd backend && go run ./cmd/server

frontend-dev:
	cd frontend && npm run dev

build:
	cd frontend && npm install --no-audit --no-fund && npm run build
	rm -rf backend/internal/webui/dist
	mkdir -p backend/internal/webui/dist
	cp -R frontend/dist/. backend/internal/webui/dist/
	cd backend && CGO_ENABLED=0 go build -ldflags="-s -w" -o ../netglance ./cmd/server

# Cross-compile a native macOS binary using Docker (no local Go toolchain needed).
build-mac: build-mac-arm64
build-mac-arm64:
	docker run --rm -v "$$PWD:/work" -w /work/frontend node:22-alpine sh -c "npm install --no-audit --no-fund && npm run build"
	rm -rf backend/internal/webui/dist && mkdir -p backend/internal/webui/dist
	cp -R frontend/dist/. backend/internal/webui/dist/
	docker run --rm -v "$$PWD:/work" -w /work/backend -e GOOS=darwin -e GOARCH=arm64 -e CGO_ENABLED=0 \
	  golang:1.23-alpine go build -trimpath -ldflags="-s -w" -o /work/netglance-darwin-arm64 ./cmd/server
	@echo "Built ./netglance-darwin-arm64"
build-mac-amd64:
	docker run --rm -v "$$PWD:/work" -w /work/frontend node:22-alpine sh -c "npm install --no-audit --no-fund && npm run build"
	rm -rf backend/internal/webui/dist && mkdir -p backend/internal/webui/dist
	cp -R frontend/dist/. backend/internal/webui/dist/
	docker run --rm -v "$$PWD:/work" -w /work/backend -e GOOS=darwin -e GOARCH=amd64 -e CGO_ENABLED=0 \
	  golang:1.23-alpine go build -trimpath -ldflags="-s -w" -o /work/netglance-darwin-amd64 ./cmd/server
	@echo "Built ./netglance-darwin-amd64"

docker:
	docker build -t netglance:dev .

test:
	cd backend && go test ./...

tidy:
	cd backend && go mod tidy

clean:
	rm -rf netglance netglance-darwin-* backend/internal/webui/dist frontend/dist
