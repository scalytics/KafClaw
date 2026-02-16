# Multi-stage build: compiles inside the container so it works cross-platform
# (Mac ARM64 host building Linux AMD64 container images).

# ---- Builder ----
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /gomikrobot ./cmd/gomikrobot

# ---- Runtime ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates git

COPY --from=builder /gomikrobot /usr/local/bin/gomikrobot
COPY web /app/web
COPY scripts/docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh

WORKDIR /app
EXPOSE 18790 18791

ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["gateway"]
