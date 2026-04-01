# Stage 1: Build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go binaries (server + agent CLI)
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/frontend/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /agent-cli ./cmd/agent-cli

# Stage 3: Minimal runtime (dashboard)
FROM alpine:3.20
RUN apk add --no-cache ca-certificates docker-cli
WORKDIR /app
COPY --from=builder /server .
COPY --from=builder /agent-cli .
EXPOSE 9222
CMD ["./server"]
