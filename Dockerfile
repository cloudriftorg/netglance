# syntax=docker/dockerfile:1.7

# Stage 1: frontend build
FROM node:22-alpine AS fe
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm install --no-audit --no-fund
COPY frontend/ ./
RUN npm run build

# Stage 2: backend build (static binary, no CGO)
FROM golang:1.23-alpine AS be
WORKDIR /src
COPY backend/go.mod backend/go.sum* ./
RUN go mod download || true
COPY backend/ ./
COPY --from=fe /app/dist ./internal/webui/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /netglance ./cmd/server
# Pre-create /data owned by the nonroot UID (65532) so a fresh named volume
# inherits the right ownership for SQLite.
RUN mkdir -p /out/data && chown 65532:65532 /out/data

# Stage 3: runtime (distroless, nonroot)
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=be /netglance /netglance
COPY --from=be --chown=nonroot:nonroot /out/data /data
EXPOSE 8080
USER nonroot:nonroot
VOLUME ["/data"]
ENTRYPOINT ["/netglance"]
