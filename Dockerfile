FROM golang:1.24.3-alpine AS builder

RUN apk add --no-cache git build-base

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/scheduler ./cmd/scheduler/main.go


FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/bin/scheduler /app/bin/scheduler
COPY --from=builder /app/internal/config ./internal/config
COPY --from=builder /app/.env .env


CMD ["/app/bin/scheduler"]