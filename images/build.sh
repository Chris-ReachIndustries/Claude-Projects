#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "Building agent-cli binary..."
MSYS_NO_PATHCONV=1 docker run --rm -v "$PROJECT_DIR://src" -w //src \
  -e CGO_ENABLED=0 -e GOOS=linux -e GOARCH=amd64 \
  golang:1.24-alpine go build -ldflags="-s -w" -o images/agent/agent-cli ./cmd/agent-cli

echo "Building claude-agent image..."
docker build -t claude-agent "$SCRIPT_DIR/agent"

echo ""
echo "Agent image built:"
docker images claude-agent --format "  {{.Repository}}:{{.Tag}} — {{.Size}}"

echo ""
echo "Cleaning up binary..."
rm -f "$SCRIPT_DIR/agent/agent-cli"
echo "Done."
