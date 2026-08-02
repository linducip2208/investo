FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o investo ./cmd/server/

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/investo .
COPY --from=builder /app/web/templates ./web/templates
COPY --from=builder /app/web/static ./web/static
COPY --from=builder /app/internal/database/migrations ./internal/database/migrations
EXPOSE 8080
CMD ["./investo"]
