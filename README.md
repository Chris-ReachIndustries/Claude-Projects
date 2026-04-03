# Claude Projects

A self-hosted platform for managing autonomous Claude agents running in Docker containers. A PM agent orchestrates specialist sub-agents, has real-time conversations with them, and produces deliverables — all visible in a live dashboard.

## Features

- **Container-Based Agents** — 7 specialized images: base (49MB), dev, go, data, doc-reader, web, printingpress
- **PM Orchestration** — PM agent spawns specialists, reads their output, has conversations, makes strategic decisions
- **162 Agent Roles** — Searchable library of specialist roles with expert system prompts
- **Real-Time Streaming** — Agent thinking, tool calls, and conversations visible live in the dashboard
- **PM Tool Enforcement** — PM restricted to orchestration tools only (can't do work itself, must delegate)
- **Inter-Agent Conversations** — PM messages agents between turns, gets instant replies via SSE
- **Session Persistence** — Agents resume with full conversation history
- **OAuth Enterprise Auth** — Works with existing Claude subscriptions via OAuth bridge
- **Next.js Dashboard** — Modern frontend with unified timeline, auth gate, SSE provider

## Quick Start

```bash
# 1. Clone
git clone https://github.com/Chris-ReachIndustries/Claude-Projects.git
cd Claude-Projects

# 2. Build agent images (7 specialized containers)
bash images/build.sh

# 3. Start backend + frontend
docker compose up -d --build

# 4. Open http://localhost:4173
```

On first launch, the setup wizard configures:
- Path to your `~/.claude` credentials (for OAuth auth)
- Workspace directory for agent projects
- Agent defaults (resource limits, max concurrent)

## Architecture

```
Docker Host
├── backend (Go API, port 9222)
│   ├── REST API + SSE event broker
│   ├── Docker spawner (manages agent containers)
│   └── SQLite database
│
├── frontend (Next.js, port 4173)
│   ├── Unified timeline (thinking + tools + messages)
│   ├── SSE real-time updates
│   └── Auth gate + role picker
│
├── cam-agent-1 (claude-agent, 49MB)
│   ├── Custom Go CLI with 30+ tools
│   ├── OAuth bridge auth
│   └── /workspace (mounted project dir)
│
├── cam-agent-2 (claude-agent-dev, 400MB)
├── cam-agent-3 (claude-agent-printingpress, 530MB)
└── ...
```

## Container Images

| Image | Size | Tools |
|-------|------|-------|
| `claude-agent` | 49MB | bash, curl, git, jq |
| `claude-agent-dev` | 400MB | Python 3 + Node.js |
| `claude-agent-go` | 431MB | Go 1.24 + build tools |
| `claude-agent-data` | 1GB | pandas, numpy, matplotlib, scikit-learn |
| `claude-agent-doc-reader` | 652MB | pdfplumber, python-docx, openpyxl |
| `claude-agent-web` | 1.9GB | Playwright + Chromium |
| `claude-agent-printingpress` | 530MB | WeasyPrint + PrintingPress PDF system |

## Project Structure

```
├── backend/
│   ├── cmd/
│   │   ├── server/         # Dashboard API server
│   │   └── agent-cli/      # Agent CLI (30+ tools, agentic loop)
│   ├── internal/
│   │   ├── api/            # HTTP handlers, SSE, auth, roles
│   │   ├── db/             # SQLite (12 entity-specific files)
│   │   ├── services/       # Spawner, backup, retention, webhooks
│   │   └── roles/          # 162 embedded agent roles
│   ├── Dockerfile
│   ├── go.mod
│   └── go.sum
├── frontend/
│   ├── src/
│   │   ├── app/            # Next.js pages (dashboard, projects, agents, settings)
│   │   ├── components/     # Unified timeline, spawn dialog, auth gate
│   │   ├── providers/      # SSE provider
│   │   ├── lib/            # API client, time utils
│   │   └── types/          # TypeScript interfaces
│   ├── Dockerfile
│   └── package.json
├── images/
│   ├── agent/              # Base agent Dockerfile
│   ├── agent-dev/          # Python + Node.js agent
│   ├── agent-go/           # Go agent
│   ├── agent-data/         # Data science agent
│   ├── agent-doc-reader/   # Document reader agent
│   ├── agent-web/          # Web scraper agent
│   ├── agent-printingpress/# PDF generator agent
│   └── build.sh            # Builds all 7 images
├── docker-compose.yml
└── README.md
```

## How It Works

1. User creates a project in the dashboard
2. PM agent spawns with restricted orchestration tools (can't do work itself)
3. PM reads input, makes strategic decisions, spawns specialist sub-agents
4. Sub-agents work in isolated project workspace, report findings back to PM
5. PM receives messages between turns (interruptible), has conversations with agents
6. PM closes agents when satisfied, spawns next wave based on findings
7. All thinking, tool calls, and conversations visible live in the timeline
8. Sub-agents auto-close after 5 minutes idle if PM doesn't respond

## License

MIT
