# Agent image for web scraping and testing with Playwright
FROM node:20-slim

RUN apt-get update -qq && \
    apt-get install -y -qq --no-install-recommends \
      bash curl git jq ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN npm install -g playwright && \
    npx playwright install chromium --with-deps

COPY agent-cli /usr/local/bin/agent-cli
RUN chmod +x /usr/local/bin/agent-cli

RUN useradd -m -d /home/agent agent
USER agent
RUN git config --global --add safe.directory /workspace && \
    git config --global user.email "agent@claude-projects.local" && \
    git config --global user.name "Claude Agent"

WORKDIR /workspace
ENTRYPOINT ["agent-cli"]
