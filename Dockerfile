FROM golang:1.26 AS builder

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o cartero ./cmd/cartero

FROM gcr.io/distroless/static-debian12

WORKDIR /app

COPY --from=builder /build/cartero /app/cartero

COPY config.sample.toml /app/config.sample.toml

COPY templates /app/templates

COPY assets /app/assets

COPY db/migrations /app/db/migrations

ENTRYPOINT ["/app/cartero"]
CMD ["-config", "/app/config.toml"]
