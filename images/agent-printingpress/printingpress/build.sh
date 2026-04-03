#!/bin/bash
# ─── PrintingPress — Docker Build Wrapper ───
#
# Usage:
#   ./build.sh <document.py>              Build a document (internal or external)
#   ./build.sh <document.py> <name>       Build with custom output name
#
# Internal documents (inside PrintingPress/):
#   ./build.sh documents/examples/reach_showcase.py
#
# External documents (from any project):
#   ~/Desktop/PrintingPress/build.sh ~/Desktop/MyProject/reports/my_report.py
#
# Output goes next to the document (in an output/ subdirectory).
# For internal documents, output goes to PrintingPress/output/ as before.
#
# Requires: Docker Desktop running

set -e

IMAGE_NAME="reach/printingpress"
PRINTINGPRESS_DIR="$(cd "$(dirname "$0")" && pwd)"

if [ -z "$1" ]; then
  echo "Usage: ./build.sh <document.py> [output_name]"
  echo ""
  echo "Available documents (internal):"
  find "$PRINTINGPRESS_DIR/documents" -name "*.py" -not -path "*/\.*" 2>/dev/null | sort
  exit 1
fi

DOC_FILE="$1"
OUTPUT_NAME="${2}"

if [ ! -f "$DOC_FILE" ]; then
  echo "Error: File '$DOC_FILE' not found"
  exit 1
fi

# Build Docker image if it doesn't exist
if ! docker image inspect "$IMAGE_NAME" > /dev/null 2>&1; then
  echo "First run — building PrintingPress Docker image..."
  echo "This may take a minute. Subsequent builds will be fast."
  echo "─────────────────────────────────"
  docker build -t "$IMAGE_NAME" "$PRINTINGPRESS_DIR"
  echo "─────────────────────────────────"
  echo "Image ready."
fi

# Resolve absolute path of the document
DOC_PATH="$(cd "$(dirname "$DOC_FILE")" && pwd)/$(basename "$DOC_FILE")"
DOC_DIR="$(dirname "$DOC_PATH")"
DOC_FILENAME="$(basename "$DOC_PATH")"

echo "Building: $DOC_PATH"
echo "─────────────────────────────────"

# Detect if document is inside PrintingPress (legacy) or external
case "$DOC_PATH" in
  "$PRINTINGPRESS_DIR"/*)
    # ── Internal document (legacy mode) ──
    RELATIVE_DOC="${DOC_PATH#$PRINTINGPRESS_DIR/}"
    MSYS_NO_PATHCONV=1 docker run --rm \
      -v "$PRINTINGPRESS_DIR:/work" \
      "$IMAGE_NAME" bash -c "
        cd /work && python /work/$RELATIVE_DOC && \
        echo 'Converting to PDF...' && \
        if [ -f /work/output/.last_build ]; then
          html=\$(cat /work/output/.last_build)
          name=\$(basename \"\$html\" .html)
          weasyprint \"\$html\" \"/work/output/\${name}.pdf\"
          echo \"  -> output/\${name}.pdf\"
          rm -f /work/output/.last_build
        else
          for html in /work/output/*.html; do
            name=\$(basename \"\$html\" .html)
            weasyprint \"\$html\" \"/work/output/\${name}.pdf\"
            echo \"  -> output/\${name}.pdf\"
          done
        fi && \
        echo 'Done.'"
    ;;
  *)
    # ── External document (dual-mount mode) ──
    # PrintingPress mounted read-only at /printingpress (engine, CSS, brands)
    # Project directory mounted at /project (document, images, output)
    MSYS_NO_PATHCONV=1 docker run --rm \
      -v "$PRINTINGPRESS_DIR:/printingpress:ro" \
      -v "$DOC_DIR:/project" \
      "$IMAGE_NAME" bash -c "
        cd /project && \
        PYTHONPATH=/printingpress python /project/$DOC_FILENAME && \
        echo 'Converting to PDF...' && \
        if [ -f /project/output/.last_build ]; then
          html=\$(cat /project/output/.last_build)
          name=\$(basename \"\$html\" .html)
          weasyprint \"\$html\" \"/project/output/\${name}.pdf\"
          echo \"  -> output/\${name}.pdf\"
          rm -f /project/output/.last_build
        else
          for html in /project/output/*.html; do
            name=\$(basename \"\$html\" .html)
            weasyprint \"\$html\" \"/project/output/\${name}.pdf\"
            echo \"  -> output/\${name}.pdf\"
          done
        fi && \
        echo 'Done.'"
    echo ""
    echo "Output: $DOC_DIR/output/"
    ;;
esac
