# Build stage
FROM golang:1.26.5-bookworm

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
COPY config/config.yaml config/

EXPOSE 8080

ENTRYPOINT ["/app/main"]
