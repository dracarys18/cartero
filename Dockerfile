FROM golang:1.24 AS builder

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=1 go build -o cartero ./cmd/cartero

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app

RUN mkdir -p /app/data

COPY --from=builder /build/cartero /app/cartero

COPY config.sample.toml /app/config.sample.toml

COPY templates /app/templates

RUN chmod +x /app/cartero

ENTRYPOINT ["/app/cartero"]
CMD ["-config", "/app/config.toml"]
