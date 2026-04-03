# Agent image for data analysis and visualization
FROM python:3.12-slim

RUN apt-get update -qq && \
    apt-get install -y -qq --no-install-recommends \
      bash curl git jq ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN pip install --no-cache-dir \
    pandas numpy scipy matplotlib seaborn \
    scikit-learn openpyxl plotly

COPY agent-cli /usr/local/bin/agent-cli
RUN chmod +x /usr/local/bin/agent-cli

RUN useradd -m -d /home/agent agent
USER agent
RUN git config --global --add safe.directory /workspace && \
    git config --global user.email "agent@claude-projects.local" && \
    git config --global user.name "Claude Agent"

WORKDIR /workspace
ENTRYPOINT ["agent-cli"]
