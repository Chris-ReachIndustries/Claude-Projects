FROM alpine:3.20
RUN apk add --no-cache bash curl git jq
RUN adduser -D -h /home/agent agent
COPY agent-cli /usr/local/bin/agent-cli
RUN chmod +x /usr/local/bin/agent-cli
USER agent
# Git safe directory for mounted workspace + git identity
RUN git config --global --add safe.directory /workspace && \
    git config --global user.email "agent@claude-projects.local" && \
    git config --global user.name "Claude Agent"
WORKDIR /workspace
ENTRYPOINT ["agent-cli"]
