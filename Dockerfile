FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

RUN adduser -D -g '' appuser

WORKDIR /build

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -ldflags='-w -s -extldflags "-static"' \
    -installsuffix cgo \
    -o dove \
    ./cmd/server

# Final stage
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/passwd

COPY --from=builder /build/dove /dove
COPY --from=builder /build/migrations /migrations

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/dove", "health"]

ENTRYPOINT ["/dove"]
