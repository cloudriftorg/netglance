# syntax=docker/dockerfile:1.7

ARG VERSION=dev

# Stage 1: frontend build
FROM node:22-alpine AS fe
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm install --no-audit --no-fund
COPY frontend/ ./
RUN npm run build

# Stage 2: backend build (static binary, no CGO)
FROM golang:1.23-alpine AS be
ARG VERSION
WORKDIR /src
COPY backend/go.mod backend/go.sum* ./
RUN go mod download || true
COPY backend/ ./
COPY --from=fe /app/dist ./internal/webui/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /netglance ./cmd/server

# Stage 3: runtime — alpine, needed for the arp-scan binary used by the
# scanner (same methodology as WatchYourLAN). Runs as root because arp-scan
# requires CAP_NET_RAW; with `network_mode: host` the container already has
# unrestricted host networking, so this isn't a meaningful privilege uplift
# vs. the previous distroless image.
FROM alpine:3.20
RUN apk add --no-cache arp-scan ca-certificates tzdata \
    && mkdir -p /data
COPY --from=be /netglance /netglance
EXPOSE 8473
VOLUME ["/data"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/netglance", "healthcheck"]
ENTRYPOINT ["/netglance"]
