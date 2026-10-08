FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/investo ./cmd/server/

FROM alpine:3.19
RUN apk add --no-cache ca-certificates mariadb-client tzdata
WORKDIR /app
RUN addgroup -S investo && adduser -S -G investo -h /app investo \
    && mkdir -p /app/data/backup \
    && chown -R investo:investo /app
COPY --from=builder --chown=investo:investo /out/investo /app/investo
USER investo
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD wget -q -O /dev/null "http://127.0.0.1:8080/health?format=json" || exit 1
CMD ["./investo"]
