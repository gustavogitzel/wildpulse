# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy workspace configuration and module definitions
COPY go.work ./
COPY pkg/go.mod pkg/
COPY apps/api/go.mod apps/api/
COPY apps/workers/go.mod apps/workers/

# Copy source code
COPY pkg/ pkg/
COPY apps/api/ apps/api/

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /wildpulse-api ./apps/api/cmd/server/main.go

# Runtime stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /wildpulse-api .

EXPOSE 8080

CMD ["./wildpulse-api"]
