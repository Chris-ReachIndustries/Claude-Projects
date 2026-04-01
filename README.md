# Claude Projects

A self-hosted platform for managing autonomous Claude Code agents running in Docker containers. Spawn, monitor, message, and coordinate multiple Claude agents from a single web interface.

## Features

- **Container-Based Agents** — Each agent runs in an isolated Docker container (~49MB)
- **Custom Go CLI** — 26 built-in tools: bash, file ops, grep, glob, web fetch, task management, agent orchestration
- **PM Orchestration** — Project Manager agents autonomously spawn and coordinate sub-agents
- **Native Agent Communication** — Agents message each other via built-in relay tool
- **SSE Real-Time Events** — Instant message delivery, no polling
- **OAuth Enterprise Auth** — Works with existing Claude subscriptions, no API key needed
- **Dashboard** — Web UI for monitoring, messaging, and managing all agents
- **Session Persistence** — Conversations continue across tasks within the same agent

## Prerequisites

- Docker and Docker Compose

## Quick Start

```bash
# 1. Clone the repo
git clone https://github.com/Chris-ReachIndustries/Claude-Projects.git
cd Claude-Projects

# 2. Build the agent image (one-time, ~49MB)
bash images/build.sh

# 3. Start the dashboard
docker compose up -d --build

# 4. Open http://localhost:9222
```

On first launch, the setup wizard configures:
- Path to your `~/.claude` credentials (for OAuth auth)
- Workspace directory for agent projects
- Agent defaults (resource limits, max concurrent)

## Architecture

```
Docker Host
├── cam (dashboard, port 9222)
│   ├── Go backend (REST API + SSE + spawner)
│   ├── React frontend (embedded via go:embed)
│   └── SQLite database
│
├── cam-agent-1 (Go CLI, ~49MB)
│   ├── 26 built-in tools
│   ├── OAuth bridge auth
│   └── /workspace (mounted project dir)
│
├── cam-agent-2 ...
└── cam-agent-3 ...
```

## Agent Tools (26 total)

| Category | Tools |
|----------|-------|
| **Execution** | bash |
| **Files** | read_file, write_file, edit_file, list_files |
| **Search** | grep, glob |
| **Web** | web_fetch, web_search |
| **Agent Comms** | relay_message, spawn_agent, list_project_agents, post_update |
| **Tasks** | task_create, task_update, task_list |
| **Interactive** | ask_user, plan_mode |
| **Scheduling** | schedule_task, cancel_schedule, list_schedules |
| **Code Intel** | find_definition, find_references |
| **Notebooks** | notebook_edit |

## Project Structure

```
├── cmd/
│   ├── server/         # Dashboard server (Go)
│   └── agent-cli/      # Agent CLI binary (Go, 26 tools)
├── internal/
│   ├── api/            # REST routes, SSE, auth
│   ├── db/             # SQLite database
│   └── services/       # Spawner, backup, retention, webhooks
├── frontend/           # React 19 + Vite + Tailwind
├── images/agent/       # Agent container Dockerfile
├── web/                # Embedded frontend (go:embed)
├── Dockerfile          # Multi-stage build
└── docker-compose.yml
```

## How It Works

1. User creates a project in the dashboard
2. PM agent spawns automatically with orchestration tools
3. PM plans work, spawns sub-agents using `spawn_agent` tool
4. Sub-agents work in shared workspace, communicate via `relay_message`
5. All activity visible in real-time via SSE events
6. Files appear on host filesystem (Docker bind mount)

## License

MIT
