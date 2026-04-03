# Agent image for reading PDFs, Word docs, Excel files, PowerPoint
FROM python:3.12-slim

RUN apt-get update -qq && \
    apt-get install -y -qq --no-install-recommends \
      bash curl git jq ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN pip install --no-cache-dir \
    pdfplumber pdfminer.six pypdfium2 \
    python-docx openpyxl pandas python-pptx

COPY agent-cli /usr/local/bin/agent-cli
RUN chmod +x /usr/local/bin/agent-cli

RUN useradd -m -d /home/agent agent
USER agent
RUN git config --global --add safe.directory /workspace && \
    git config --global user.email "agent@claude-projects.local" && \
    git config --global user.name "Claude Agent"

WORKDIR /workspace
ENTRYPOINT ["agent-cli"]
