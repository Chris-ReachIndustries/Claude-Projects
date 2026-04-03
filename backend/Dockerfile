# Build Go server binary
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server

# Minimal runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates docker-cli
WORKDIR /app
COPY --from=builder /server .
EXPOSE 9222
CMD ["./server"]
