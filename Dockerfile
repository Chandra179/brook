# Build stage
FROM golang:1.27.0-bookworm AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/main ./cmd/example/

# Run stage - using Alpine for very small image
FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/main .
COPY config/config_prd.yaml config/
COPY config/config_dev.yaml config/

EXPOSE 8080

ENV APP_ENVIRONMENT=prd

ENTRYPOINT ["/app/main"]
