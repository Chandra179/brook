# Build stage
FROM golang:1.26.4-bookworm@sha256:5f68ec6805843bd3981a951ffada82a26a0bd2631045c8f7dba483fa868f5ec5 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/main ./cmd/example/

# Run stage - using Alpine for very small image
FROM alpine:3.21@sha256:48b0309ca019d89d40f670aa1bc06e426dc0931948452e8491e3d65087abc07d

RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/main .
COPY config/config.yaml config/

EXPOSE 8080

ENTRYPOINT ["/app/main"]
