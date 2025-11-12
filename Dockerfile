FROM golang:1.24.3-alpine AS builder

RUN apk add --no-cache git build-base

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/scheduler ./cmd/scheduler/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/worker ./worker/cmd/worker/main.go


FROM alpine:latest

RUN apk add --no-cache ca-certificates postgresql-client

WORKDIR /app

COPY --from=builder /app/bin/scheduler /app/bin/scheduler
COPY --from=builder /app/bin/worker /app/bin/worker
COPY --from=builder /app/internal/config ./internal/config
COPY --from=builder /app/worker/internal/config ./worker/internal/config
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/scripts/migrate.sh ./scripts/migrate.sh

RUN chmod +x ./scripts/migrate.sh

CMD ["/app/bin/scheduler"]