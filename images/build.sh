#!/bin/bash
# Build all agent Docker images with a shared agent-cli binary
# Usage: ./images/build.sh [image-name...]
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

ALL_IMAGES="agent agent-dev agent-go agent-data agent-doc-reader agent-web agent-printingpress"

if [ $# -gt 0 ]; then IMAGES="$@"; else IMAGES="$ALL_IMAGES"; fi

echo "=== Building agent-cli ==="
MSYS_NO_PATHCONV=1 docker run --rm \
  -v "$PROJECT_DIR:/app" -w /app \
  -v "$SCRIPT_DIR/agent:/out" \
  golang:1.24-alpine sh -c "go build -ldflags='-s -w' -o /out/agent-cli ./cmd/agent-cli"

for img in $IMAGES; do
  [ "$img" != "agent" ] && [ -d "$SCRIPT_DIR/$img" ] && cp "$SCRIPT_DIR/agent/agent-cli" "$SCRIPT_DIR/$img/agent-cli"
done

echo "=== Building images ==="
for img in $IMAGES; do
  [ ! -d "$SCRIPT_DIR/$img" ] && echo "SKIP $img" && continue
  echo "claude-$img..."
  docker build --no-cache -t "claude-$img" "$SCRIPT_DIR/$img" | tail -1
done

for img in $IMAGES; do
  [ "$img" != "agent" ] && rm -f "$SCRIPT_DIR/$img/agent-cli"
done

echo "=== Done ==="
docker images | grep claude-agent | awk '{printf "%-40s %s\n", $1, $NF}'
