FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git gcc musl-dev sqlite-dev

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o cartero ./cmd/cartero

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata sqlite-libs

WORKDIR /app

RUN mkdir -p /app/data

COPY --from=builder /build/cartero /app/cartero

COPY config.sample.toml /app/config.sample.toml

RUN chmod +x /app/cartero && \
    chown -R nobody:nobody /app

USER nobody

ENTRYPOINT ["/app/cartero"]
CMD ["-config", "/app/config.toml"]
