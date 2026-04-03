# PrintingPress — Docker Build Image
#
# Pre-builds all dependencies so individual document builds are fast.
# Includes mermaid-cli (mmdc) for rendering mermaid diagrams via puppeteer.
#
# Build once:  docker build -t printingpress .
# Then use:    ./build.sh documents/my_doc.py

FROM python:3.12-slim

# System deps for WeasyPrint rendering + Chromium for mermaid-cli
RUN apt-get update -qq && \
    apt-get install -y -qq --no-install-recommends \
      libpango-1.0-0 libpangocairo-1.0-0 libcairo2 \
      libglib2.0-0 libharfbuzz0b libfontconfig1 \
      shared-mime-info fonts-liberation fonts-dejavu-core fontconfig \
      curl ca-certificates \
      chromium \
      nodejs npm && \
    fc-cache -f && \
    rm -rf /var/lib/apt/lists/*

# Python packages
RUN pip install --no-cache-dir weasyprint cairosvg pillow

# mermaid-cli — uses real mermaid.js via puppeteer/chromium
ENV PUPPETEER_SKIP_CHROMIUM_DOWNLOAD=true
ENV PUPPETEER_EXECUTABLE_PATH=/usr/bin/chromium
RUN npm install -g @mermaid-js/mermaid-cli

WORKDIR /work
