# Build stage
FROM golang:1.26.5-bookworm AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/main ./cmd/example/

# Run stage - using Alpine for very small image
FROM alpine:3.21

RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/main .
COPY config/config_prd.yaml config/
COPY config/config_dev.yaml config/

EXPOSE 8080

ENV APP_ENVIRONMENT=prd

ENTRYPOINT ["/app/main"]
