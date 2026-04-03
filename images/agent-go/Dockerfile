# Agent image for Go development
FROM golang:1.24-alpine

RUN apk add --no-cache bash curl git jq make

COPY agent-cli /usr/local/bin/agent-cli
RUN chmod +x /usr/local/bin/agent-cli

RUN adduser -D -h /home/agent agent
USER agent
RUN git config --global --add safe.directory /workspace && \
    git config --global user.email "agent@claude-projects.local" && \
    git config --global user.name "Claude Agent"

WORKDIR /workspace
ENTRYPOINT ["agent-cli"]
