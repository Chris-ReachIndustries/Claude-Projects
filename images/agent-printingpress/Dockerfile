# Agent image with PrintingPress PDF generation capability
# Based on PrintingPress container + agent CLI

FROM python:3.12-slim

# System deps for WeasyPrint + Chromium
RUN apt-get update -qq && \
    apt-get install -y -qq --no-install-recommends \
      libpango-1.0-0 libpangocairo-1.0-0 libcairo2 \
      libglib2.0-0 libharfbuzz0b libfontconfig1 \
      shared-mime-info fonts-liberation fonts-dejavu-core fontconfig \
      curl ca-certificates git jq bash \
      && fc-cache -f && rm -rf /var/lib/apt/lists/*

# Python packages for WeasyPrint
RUN pip install --no-cache-dir weasyprint cairosvg pillow

# Copy PrintingPress build system
COPY printingpress/ /opt/printingpress/

# Copy agent CLI
COPY agent-cli /usr/local/bin/agent-cli
RUN chmod +x /usr/local/bin/agent-cli

# Non-root user
RUN useradd -m -d /home/agent agent
USER agent
RUN git config --global --add safe.directory /workspace && \
    git config --global user.email "agent@claude-projects.local" && \
    git config --global user.name "Claude Agent"

WORKDIR /workspace
ENTRYPOINT ["agent-cli"]
