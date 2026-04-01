FROM alpine:3.20
RUN apk add --no-cache bash curl git jq
RUN adduser -D -h /home/agent agent
COPY agent-cli /usr/local/bin/agent-cli
RUN chmod +x /usr/local/bin/agent-cli
USER agent
WORKDIR /workspace
ENTRYPOINT ["agent-cli"]
