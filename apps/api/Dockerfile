# ── Stage 1: Build ──────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

WORKDIR /workspace

# ── Step 1: Download dependencies
COPY go.work go.work
COPY go.work.sum go.work.sum
COPY apps/shared/go.mod  apps/shared/go.mod
COPY apps/shared/go.sum  apps/shared/go.sum
COPY apps/api/go.mod     apps/api/go.mod
COPY apps/api/go.sum     apps/api/go.sum
# go.work references all three modules — agent go.mod must exist too
COPY apps/agent/go.mod   apps/agent/go.mod
COPY apps/agent/go.sum   apps/agent/go.sum

RUN cd apps/api && go mod download

# ── Step 2: Copy actual source code
COPY apps/shared/ apps/shared/
COPY apps/api/    apps/api/

# ── Step 3: Build the binary
WORKDIR /workspace/apps/api
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /bin/server ./cmd/server

# ── Stage 2: Runtime ────────────────────────────────────────────────────────
FROM alpine:latest

# Copy CA certs for HTTPS calls
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /bin/server /server

EXPOSE 8080

ENTRYPOINT ["/server"]