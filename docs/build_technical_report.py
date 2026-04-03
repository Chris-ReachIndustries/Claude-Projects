"""
Claude Agent Manager (Claude Projects) — Technical Report
Reach Industries, April 2026

Build with:
    ~/Desktop/PrintingPress/build.sh ~/Desktop/Claude_Agent_Manager/docs/build_technical_report.py
"""
import sys, os
sys.path.insert(0, os.path.join(os.path.expanduser('~'), 'Desktop', 'PrintingPress'))
from build import build_document

# ─── Report Content ──────────────────────────────────────────────────────────

CONTENT = """

<!-- ════════════════════════════════════════════════════════════════════════ -->
<!-- SECTION 1: SYSTEM OVERVIEW                                              -->
<!-- ════════════════════════════════════════════════════════════════════════ -->

<div class="page">
  <div class="h1">1. System Overview</div>

  <div class="h2">1.1 What is Claude Agent Manager?</div>
  <p class="p">
    Claude Agent Manager (published as <strong>Claude Projects</strong>) is a self-hosted platform
    for orchestrating autonomous AI agents powered by Anthropic&rsquo;s Claude models. It provides
    a complete infrastructure for spawning, managing, and monitoring Claude agents that run inside
    isolated Docker containers, coordinated by a lead Project Manager (PM) agent that delegates
    work to specialist sub-agents.
  </p>
  <p class="p">
    The system transforms the single-user, single-session Claude CLI experience into a
    multi-agent orchestration platform where a PM agent autonomously decomposes complex
    projects into discrete tasks, selects specialists from a library of 162 pre-defined
    roles, spawns them in purpose-built containers, conducts real-time conversations with
    them, reviews their output, and iterates until deliverables meet quality standards.
  </p>
  <p class="p">
    All agent activity &mdash; thinking, tool calls, inter-agent messages, file creation,
    and strategic decisions &mdash; streams live to a Next.js dashboard via Server-Sent Events (SSE),
    giving the user full visibility into the orchestration process without needing to intervene.
  </p>

  <div class="h2">1.2 The Problem It Solves</div>
  <p class="p">
    Large language models are powerful but fundamentally single-threaded in their default interfaces.
    A single Claude session can research, write, or code &mdash; but it cannot simultaneously run
    a market researcher, a data analyst, a copywriter, and a technical architect, then synthesise
    their outputs into a coherent strategy. Claude Agent Manager solves this by providing:
  </p>
  <ul class="ul">
    <li><strong>Parallel execution</strong> &mdash; multiple agents work simultaneously in isolated containers,
        each with their own conversation context and tooling</li>
    <li><strong>Specialist delegation</strong> &mdash; a PM agent selects the right specialist for each task
        from a 162-role library, matching container images to requirements (data science agents get
        pandas/matplotlib, web agents get Playwright/Chromium, PDF agents get WeasyPrint)</li>
    <li><strong>Context isolation</strong> &mdash; each agent operates in its own container with a focused
        system prompt and task description, avoiding the context pollution that degrades quality
        in long single-agent sessions</li>
    <li><strong>Persistent orchestration</strong> &mdash; the PM agent maintains a continuous conversation
        thread across dozens of sub-agent interactions, building on findings rather than starting fresh</li>
    <li><strong>Enterprise auth compatibility</strong> &mdash; works with existing Claude Pro/Team/Enterprise
        subscriptions via OAuth bridge discovery, requiring no separate API key purchase</li>
  </ul>

  <div class="h2">1.3 Target Users</div>
  <p class="p">
    Claude Projects is designed for technical teams and individual power users who need to:
  </p>
  <ul class="ul">
    <li>Execute complex, multi-phase projects that require diverse specialist knowledge</li>
    <li>Produce multiple coordinated deliverables (reports, code, analyses, documentation)</li>
    <li>Maintain quality control over AI-generated output through PM review cycles</li>
    <li>Run AI workloads in controlled, isolated environments with resource limits</li>
    <li>Leverage existing Claude subscriptions without additional API cost management</li>
  </ul>
  <p class="p">
    The self-hosted architecture (Docker Compose on a local machine or server) appeals to
    users who want full control over their agent infrastructure, data, and credentials.
  </p>
</div>

<!-- ════════════════════════════════════════════════════════════════════════ -->
<!-- SECTION 2: ARCHITECTURE                                                 -->
<!-- ════════════════════════════════════════════════════════════════════════ -->

<div class="page">
  <div class="h1">2. Architecture</div>

  <p class="p">
    The system consists of four major components: a Go API backend, a Next.js frontend,
    a custom Go agent CLI binary, and a fleet of 7 specialized Docker container images.
    These communicate via REST APIs, Server-Sent Events, and a shared Docker-mounted workspace.
  </p>

  <div class="h2">2.1 Backend: Go API Server</div>
  <p class="p">
    The backend is a single Go binary (<code>cmd/server/main.go</code>) that serves as the
    central coordination hub. It runs on port 9222 inside a Docker container with access to
    the Docker socket for spawning agent containers.
  </p>

  <div class="h3">Core Components</div>
  <table class="table">
    <tr>
      <th style="width:25%">Component</th>
      <th style="width:35%">Package</th>
      <th style="width:40%">Responsibility</th>
    </tr>
    <tr>
      <td>HTTP Router</td>
      <td><code>internal/api/router.go</code></td>
      <td>70+ REST endpoints using Go 1.22 enhanced ServeMux with method routing</td>
    </tr>
    <tr>
      <td>SSE Broker</td>
      <td><code>internal/api/sse.go</code></td>
      <td>Real-time event broadcasting to dashboard clients (max 10 concurrent)</td>
    </tr>
    <tr>
      <td>Container Spawner</td>
      <td><code>internal/services/spawner.go</code></td>
      <td>Channel-driven Docker container lifecycle management</td>
    </tr>
    <tr>
      <td>Database</td>
      <td><code>internal/db/</code></td>
      <td>SQLite with WAL mode, 12 entity-specific files, triggers, migrations</td>
    </tr>
    <tr>
      <td>Auth Middleware</td>
      <td><code>internal/api/auth.go</code></td>
      <td>API key authentication with setup-wizard bypass</td>
    </tr>
    <tr>
      <td>Role Library</td>
      <td><code>internal/roles/</code></td>
      <td>162 embedded agent roles with system prompts, loaded via <code>go:embed</code></td>
    </tr>
  </table>

  <div class="h3">API Surface</div>
  <p class="p">
    The router exposes a comprehensive REST API organised into resource groups:
  </p>
  <ul class="ul">
    <li><strong>Agents</strong> (14 endpoints) &mdash; CRUD, updates, messages, relay, files, export, close, resume</li>
    <li><strong>Projects</strong> (11 endpoints) &mdash; CRUD, start/pause/complete lifecycle, spawn agents, unified timeline</li>
    <li><strong>Launch Requests</strong> (3 endpoints) &mdash; queue-based container spawn requests</li>
    <li><strong>Workflows</strong> (5 endpoints) &mdash; multi-step automated workflows</li>
    <li><strong>Webhooks</strong> (4 endpoints) &mdash; external notification integration</li>
    <li><strong>Push</strong> (3 endpoints) &mdash; Web Push notification subscriptions</li>
    <li><strong>Retention</strong> (3 endpoints) &mdash; automated data cleanup</li>
    <li><strong>Settings</strong> (3 endpoints) &mdash; setup wizard and configuration</li>
    <li><strong>Roles</strong> (3 endpoints) &mdash; agent role library search and retrieval</li>
    <li><strong>Workspace</strong> (2 endpoints) &mdash; project file browsing</li>
    <li><strong>SSE</strong> (1 endpoint) &mdash; real-time event stream</li>
    <li><strong>Health</strong> (2 endpoints) &mdash; server and database health checks</li>
  </ul>
  <p class="p">
    Middleware is stacked as: CORS &rarr; Auth &rarr; Gzip &rarr; Router. SSE connections are
    explicitly excluded from gzip compression to maintain streaming semantics.
  </p>

  <div class="h3">Background Services</div>
  <p class="p">
    The server starts several goroutines alongside the HTTP listener:
  </p>
  <ul class="ul">
    <li><strong>Container Spawner</strong> &mdash; processes launch request queue via channel notification</li>
    <li><strong>Container Health Checker</strong> &mdash; polls Docker every 2 minutes for crashed containers</li>
    <li><strong>Auto-Archive Sweep</strong> &mdash; archives agents inactive for 30+ minutes every 5 minutes</li>
    <li><strong>Backup Scheduler</strong> &mdash; periodic SQLite database backups</li>
    <li><strong>Retention Service</strong> &mdash; automated cleanup of old data based on retention policies</li>
  </ul>
</div>

<div class="page">
  <div class="h2">2.2 Database: SQLite with WAL</div>
  <p class="p">
    The database layer uses SQLite in WAL (Write-Ahead Logging) mode, split across 12
    entity-specific Go files in <code>internal/db/</code>. The schema defines 10 tables
    with referential integrity, computed field triggers, and indexed foreign keys.
  </p>

  <div class="h3">Schema Overview</div>
  <table class="table">
    <tr>
      <th style="width:22%">Table</th>
      <th style="width:18%">Key Columns</th>
      <th style="width:60%">Purpose</th>
    </tr>
    <tr>
      <td><code>agents</code></td>
      <td>23 columns</td>
      <td>Agent records with status, project linkage, parent-child relationships,
          computed fields (pending messages, unread updates, latest summary)</td>
    </tr>
    <tr>
      <td><code>updates</code></td>
      <td>6 columns</td>
      <td>Agent activity timeline: text, thinking, tool calls, errors, status changes</td>
    </tr>
    <tr>
      <td><code>messages</code></td>
      <td>8 columns</td>
      <td>Inter-agent and user-agent messages with delivery tracking (pending/delivered/acknowledged/executed)</td>
    </tr>
    <tr>
      <td><code>files</code></td>
      <td>9 columns</td>
      <td>Uploaded file metadata with disk-based storage (migrated from BLOB)</td>
    </tr>
    <tr>
      <td><code>projects</code></td>
      <td>11 columns</td>
      <td>Project records with PM agent linkage, concurrency limits, autonomy mode</td>
    </tr>
    <tr>
      <td><code>project_updates</code></td>
      <td>4 columns</td>
      <td>Project-level timeline events (milestones, decisions, info, errors)</td>
    </tr>
    <tr>
      <td><code>launch_requests</code></td>
      <td>10 columns</td>
      <td>Container spawn queue: new, resume, terminate requests with lifecycle tracking</td>
    </tr>
    <tr>
      <td><code>settings</code></td>
      <td>2 columns</td>
      <td>Key-value configuration store (workspace root, auth paths, resource limits)</td>
    </tr>
    <tr>
      <td><code>webhooks</code></td>
      <td>6 columns</td>
      <td>External webhook notification targets with failure tracking</td>
    </tr>
    <tr>
      <td><code>workflows</code></td>
      <td>8 columns</td>
      <td>Multi-step automated workflow definitions and execution state</td>
    </tr>
  </table>

  <div class="h3">Triggers and Computed Fields</div>
  <p class="p">
    Two SQLite triggers maintain denormalized counters on the agents table for dashboard
    performance. <code>after_message_insert</code> increments <code>pending_message_count</code>
    and caches the latest message. <code>after_update_insert</code> increments
    <code>unread_update_count</code> and caches the latest summary. These avoid expensive
    JOIN queries when listing agents in the dashboard.
  </p>

  <div class="h3">Migration Strategy</div>
  <p class="p">
    Migrations are additive-only <code>ALTER TABLE ADD COLUMN</code> statements that ignore
    &ldquo;duplicate column&rdquo; errors. This allows the schema to evolve without versioning
    or rollback support. One exception: the <code>updates</code> table CHECK constraint was
    removed by recreating the table (SQLite lacks <code>ALTER CHECK</code>), migrating data
    from the old table to the new one.
  </p>
</div>

<div class="page">
  <div class="h2">2.3 Frontend: Next.js Dashboard</div>
  <p class="p">
    The frontend is a Next.js application (App Router) running on port 4173, built with
    TypeScript and Tailwind CSS. It provides the user-facing dashboard for creating projects,
    monitoring agents, and viewing real-time activity.
  </p>

  <div class="h3">Key Components</div>
  <table class="table">
    <tr>
      <th style="width:30%">Component</th>
      <th style="width:70%">Responsibility</th>
    </tr>
    <tr>
      <td><code>SSEProvider</code></td>
      <td>React Context that manages a single EventSource connection to <code>/api/events</code>,
          dispatches typed events to subscribers via a pub-sub pattern with automatic reconnection (3s backoff)</td>
    </tr>
    <tr>
      <td><code>UnifiedTimeline</code></td>
      <td>Merges agent updates, messages, and project updates into a single chronologically-sorted
          feed with type-specific rendering (thinking blocks as violet cards, tool calls as compact
          monospace lines, messages as chat bubbles, project events as colored badges)</td>
    </tr>
    <tr>
      <td><code>AuthGate</code></td>
      <td>Guards all routes, prompts for API key on first visit, stores in localStorage</td>
    </tr>
    <tr>
      <td>Spawn Dialog</td>
      <td>Role picker with 162-role searchable library, container image selection, task input</td>
    </tr>
  </table>

  <div class="h3">SSE Event Types</div>
  <p class="p">
    The frontend subscribes to 10 SSE event types:
  </p>
  <ul class="ul">
    <li><code>agent-updated</code> / <code>agent-deleted</code> &mdash; agent state changes</li>
    <li><code>message-queued</code> &mdash; new message for an agent</li>
    <li><code>project-created</code> / <code>project-updated</code> / <code>project-deleted</code> &mdash; project lifecycle</li>
    <li><code>launch-request-created</code> / <code>launch-request-updated</code> &mdash; container spawn progress</li>
    <li><code>container-health</code> &mdash; crashed container notifications</li>
    <li><code>shutdown</code> &mdash; server restart signal (triggers reconnect, not exit)</li>
  </ul>

  <div class="h3">Unified Timeline Design</div>
  <p class="p">
    The <code>UnifiedTimeline</code> component is the primary interface for understanding what
    agents are doing. It accepts three optional data sources &mdash; agent updates, messages, and
    project updates &mdash; merges them by timestamp, and renders each entry type distinctly:
  </p>
  <ul class="ul">
    <li><strong>Thinking</strong> &mdash; violet background card with brain icon, italic text, truncated to 500 chars</li>
    <li><strong>Tool calls</strong> &mdash; compact monospace line showing tool name + parsed argument summary</li>
    <li><strong>Status</strong> &mdash; minimal grey dot with text</li>
    <li><strong>Errors</strong> &mdash; red background with alert icon</li>
    <li><strong>Text/Info</strong> &mdash; grey card with formatted markdown (bold, italic, code, bullet lists)</li>
    <li><strong>Messages</strong> &mdash; chat bubble layout (user messages right-aligned in primary colour,
        agent messages left-aligned with source agent label)</li>
    <li><strong>Project events</strong> &mdash; coloured badges by type (milestone=green, decision=purple,
        info=blue, error=red)</li>
  </ul>
  <p class="p">
    The timeline auto-scrolls to the bottom as new entries arrive, providing a live-feed
    experience of agent activity.
  </p>
</div>

<div class="page">
  <div class="h2">2.4 Agent CLI: Custom Go Binary</div>
  <p class="p">
    The agent CLI (<code>cmd/agent-cli/</code>) is a custom Go binary compiled to ~8MB that runs
    inside every agent container. It implements a complete agentic loop with 30+ tools, session
    persistence, OAuth authentication, dashboard integration, and inter-agent communication.
  </p>

  <div class="h3">Architecture of the CLI</div>
  <table class="table">
    <tr>
      <th style="width:22%">File</th>
      <th style="width:78%">Responsibility</th>
    </tr>
    <tr>
      <td><code>main.go</code></td>
      <td>Entry point: flag parsing, authentication, mode selection (agent vs single-prompt)</td>
    </tr>
    <tr>
      <td><code>api.go</code></td>
      <td>Anthropic SDK client wrapper, tool definitions (30+ tools), response extraction</td>
    </tr>
    <tr>
      <td><code>loop.go</code></td>
      <td>Agentic loop (200 iterations max), PM tool restriction, inter-turn message injection,
          agent mode with continuous conversation, sub-agent auto-close protocol</td>
    </tr>
    <tr>
      <td><code>auth.go</code></td>
      <td>OAuth bridge discovery: reads <code>.credentials.json</code>, creates code session,
          acquires bridge worker JWT. Falls back to <code>ANTHROPIC_API_KEY</code></td>
    </tr>
    <tr>
      <td><code>session.go</code></td>
      <td>Conversation persistence: JSON serialization of system prompt + message history,
          agent-ID-keyed file storage, resume on container restart</td>
    </tr>
    <tr>
      <td><code>dashboard.go</code></td>
      <td>Dashboard client: registration, update posting, message polling, file upload,
          relay messaging, health checks</td>
    </tr>
    <tr>
      <td><code>sse.go</code></td>
      <td>SSE client: connects to backend event stream, filters for messages addressed to this
          agent, delivers to channel for inter-turn injection. Exponential backoff reconnection.</td>
    </tr>
    <tr>
      <td><code>tools_*.go</code></td>
      <td>Tool implementations: file operations, bash execution, search (grep/glob), web
          fetch/search, git operations, code intelligence, task management, scheduling, scratchpad</td>
    </tr>
  </table>

  <div class="h3">Tool Categories (30+ Tools)</div>
  <table class="table">
    <tr>
      <th style="width:22%">Category</th>
      <th style="width:78%">Tools</th>
    </tr>
    <tr>
      <td>File Operations</td>
      <td><code>bash</code>, <code>read_file</code>, <code>write_file</code>, <code>edit_file</code>, <code>list_files</code></td>
    </tr>
    <tr>
      <td>Search</td>
      <td><code>grep</code>, <code>glob</code>, <code>find_definition</code>, <code>find_references</code></td>
    </tr>
    <tr>
      <td>Orchestration</td>
      <td><code>spawn_agent</code>, <code>relay_message</code>, <code>list_project_agents</code>,
          <code>close_agent</code>, <code>post_update</code>, <code>search_roles</code></td>
    </tr>
    <tr>
      <td>Collaboration</td>
      <td><code>scratchpad_write</code>, <code>scratchpad_read</code>, <code>ask_user</code>,
          <code>plan_mode</code></td>
    </tr>
    <tr>
      <td>Web</td>
      <td><code>web_fetch</code>, <code>web_search</code></td>
    </tr>
    <tr>
      <td>Git</td>
      <td><code>git_status</code>, <code>git_diff</code>, <code>git_commit</code>, <code>git_log</code></td>
    </tr>
    <tr>
      <td>Task Management</td>
      <td><code>task_create</code>, <code>task_update</code>, <code>task_list</code></td>
    </tr>
    <tr>
      <td>Scheduling</td>
      <td><code>schedule_task</code>, <code>cancel_schedule</code>, <code>list_schedules</code></td>
    </tr>
    <tr>
      <td>Notebooks</td>
      <td><code>notebook_edit</code></td>
    </tr>
  </table>
</div>

<div class="page">
  <div class="h2">2.5 Container Images: 7 Specialized Types</div>
  <p class="p">
    Every agent runs inside a Docker container built from one of 7 purpose-built images.
    Each image includes the <code>agent-cli</code> binary plus the specific tooling needed
    for its specialisation. The build process (<code>images/build.sh</code>) cross-compiles
    the Go CLI binary using <code>golang:1.24-alpine</code>, then copies it into each image.
  </p>

  <table class="table">
    <tr>
      <th style="width:25%">Image</th>
      <th style="width:10%">Size</th>
      <th style="width:15%">Base</th>
      <th style="width:50%">Included Tools</th>
    </tr>
    <tr>
      <td><code>claude-agent</code></td>
      <td>49MB</td>
      <td>Alpine 3.20</td>
      <td>bash, curl, git, jq. The minimal base for research, writing, planning, analysis.</td>
    </tr>
    <tr>
      <td><code>claude-agent-dev</code></td>
      <td>400MB</td>
      <td>Python 3.12 + Node.js</td>
      <td>Full development environment. For coding, scripting, and general software development.</td>
    </tr>
    <tr>
      <td><code>claude-agent-go</code></td>
      <td>431MB</td>
      <td>Go 1.24 Alpine</td>
      <td>Go compiler + build tools + make. For Go development and compilation.</td>
    </tr>
    <tr>
      <td><code>claude-agent-data</code></td>
      <td>~1GB</td>
      <td>Python 3.12 Slim</td>
      <td>pandas, numpy, scipy, matplotlib, seaborn, scikit-learn, plotly, openpyxl. For data
          analysis, visualisation, CSV processing, statistical modelling.</td>
    </tr>
    <tr>
      <td><code>claude-agent-doc-reader</code></td>
      <td>652MB</td>
      <td>Python 3.12 Slim</td>
      <td>pdfplumber, pdfminer, pypdfium2, python-docx, openpyxl, python-pptx, pandas. For
          reading PDFs, Word docs, Excel files, PowerPoint presentations.</td>
    </tr>
    <tr>
      <td><code>claude-agent-web</code></td>
      <td>1.9GB</td>
      <td>Node.js 20 Slim</td>
      <td>Playwright + Chromium. For web scraping, automated testing, screenshots, browser
          automation.</td>
    </tr>
    <tr>
      <td><code>claude-agent-printingpress</code></td>
      <td>530MB</td>
      <td>Python 3.12 Slim</td>
      <td>WeasyPrint, cairosvg, Pillow + bundled PrintingPress build system at
          <code>/opt/printingpress/</code>. For professional branded PDF report generation.</td>
    </tr>
  </table>

  <p class="p">
    All images share a common pattern: non-root <code>agent</code> user, git configured with
    safe directory for <code>/workspace</code>, and <code>agent-cli</code> as the entrypoint.
    The workspace directory is bind-mounted from the host at container creation time, ensuring
    all agents in a project share the same filesystem view.
  </p>

  <div class="h2">2.6 Communication Architecture</div>
  <p class="p">
    The system uses three communication channels:
  </p>

  <div class="h3">REST API (Agent &harr; Backend)</div>
  <p class="p">
    Agent containers communicate with the backend via HTTP REST calls to
    <code>http://host.docker.internal:9222</code>. The dashboard client in each agent
    (<code>dashboard.go</code>) posts updates, polls messages, uploads files, and manages
    relay messaging. All requests include the <code>DASHBOARD_API_KEY</code> for authentication.
  </p>

  <div class="h3">SSE (Backend &rarr; Frontend and Backend &rarr; Agent)</div>
  <p class="p">
    The SSE broker broadcasts typed events to all connected clients. Both the frontend and
    agent CLI maintain persistent SSE connections to <code>/api/events</code>. Agents filter
    events by their own ID to receive targeted messages. The broker supports up to 10 concurrent
    clients with 30-second keepalive pings.
  </p>

  <div class="h3">Relay Messages (Agent &harr; Agent via Backend)</div>
  <p class="p">
    Inter-agent communication uses the <code>relay_message</code> tool, which POSTs to the
    backend&rsquo;s relay endpoint. The backend stores the message in the <code>messages</code>
    table and broadcasts a <code>message-queued</code> SSE event. The target agent receives it
    either via SSE (instant) or the next poll cycle (10-second fallback). The <code>wait_for_reply</code>
    option blocks the sender for up to 2 minutes waiting for a response.
  </p>
</div>

<!-- ════════════════════════════════════════════════════════════════════════ -->
<!-- SECTION 3: HOW IT WORKS                                                 -->
<!-- ════════════════════════════════════════════════════════════════════════ -->

<div class="page">
  <div class="h1">3. How It Works</div>

  <div class="h2">3.1 Project Creation and PM Spawn</div>
  <p class="p">
    The orchestration lifecycle begins when a user creates a project in the dashboard:
  </p>
  <ol class="ol">
    <li>User provides a project name, description, workspace folder, and concurrency limit (default: 4 agents)</li>
    <li>Backend creates the project record and auto-generates a URL-safe folder slug</li>
    <li>User clicks &ldquo;Start Project&rdquo; with an optional initial prompt</li>
    <li>Backend generates a comprehensive PM system prompt via <code>generatePMPrompt()</code></li>
    <li>A launch request is created with type <code>new</code> and project metadata (PM prompt, user description)</li>
    <li>The spawner is notified via its channel and immediately processes the request</li>
    <li>A Docker container is spawned with the <code>claude-agent</code> image, project workspace mounted at
        <code>/workspace</code>, and environment variables for project ID, agent URL, and auth credentials</li>
    <li>The container starts <code>agent-cli</code>, which registers with the backend, receives any pending
        messages, and enters the agentic loop with the PM system prompt and user&rsquo;s project description</li>
  </ol>

  <div class="h2">3.2 PM System Prompt Design</div>
  <p class="p">
    The PM agent receives a carefully crafted system prompt (generated by <code>generatePMPrompt()</code>
    in <code>projects.go</code>) that defines its operational framework:
  </p>
  <ul class="ul">
    <li><strong>Identity</strong> &mdash; &ldquo;You are the lead strategist managing project: [name]&rdquo;</li>
    <li><strong>Available tools</strong> &mdash; only orchestration tools (spawn, relay, read, close, post_update,
        scratchpad, list_agents, search_roles, ask_user, tasks)</li>
    <li><strong>Container image guide</strong> &mdash; which image to select for which task type</li>
    <li><strong>Agent lifecycle protocol</strong> &mdash; 6-step cycle: spawn &rarr; wait &rarr; read output &rarr;
        have 2-3 exchanges &rarr; more work or close &rarr; never leave idle</li>
    <li><strong>Quality standards</strong> &mdash; &ldquo;Read every agent&rsquo;s output and react to it&rdquo;,
        &ldquo;Quality over quantity&rdquo;, &ldquo;Make hard calls&rdquo;</li>
    <li><strong>Retry limits</strong> &mdash; maximum 3 attempts per task, tracked via scratchpad</li>
    <li><strong>Completion protocol</strong> &mdash; verify deliverables exist, read key outputs, post final summary</li>
    <li><strong>Autonomy mode</strong> &mdash; either &ldquo;AUTONOMOUS&rdquo; (no user approval needed) or
        &ldquo;SUPERVISED&rdquo; (ask_user before major decisions)</li>
  </ul>

  <div class="h2">3.3 PM Tool Restriction and Delegation Enforcement</div>
  <p class="p">
    A critical design decision is that PM agents are restricted to orchestration tools only.
    The enforcement is two-layered:
  </p>
  <ol class="ol">
    <li><strong>Tool filtering at API call time</strong> &mdash; when the PM&rsquo;s agentic loop calls
        <code>BuildTools()</code>, only tools in the <code>pmAllowedTools</code> whitelist are sent
        to the Anthropic API. Claude never sees the work tools (bash, write_file, etc.) in its
        available tool list.</li>
    <li><strong>Runtime rejection</strong> &mdash; if the model somehow attempts a blocked tool call,
        <code>executeTools()</code> checks the whitelist again and returns an error:
        &ldquo;ERROR: As PM, you cannot use [tool] directly. Spawn a sub-agent to do this work.&rdquo;</li>
  </ol>
  <p class="p">
    The 13 PM-allowed tools are: <code>spawn_agent</code>, <code>relay_message</code>,
    <code>list_project_agents</code>, <code>post_update</code>, <code>close_agent</code>,
    <code>read_file</code>, <code>list_files</code>, <code>scratchpad_write</code>,
    <code>scratchpad_read</code>, <code>task_create</code>, <code>task_update</code>,
    <code>task_list</code>, <code>ask_user</code>, and <code>search_roles</code>.
    Note that <code>read_file</code> and <code>list_files</code> are included so the PM can
    review sub-agent output without delegating the review itself.
  </p>
</div>

<div class="page">
  <div class="h2">3.4 Sub-Agent Orchestration</div>
  <p class="p">
    When the PM decides to spawn a sub-agent, the following sequence occurs:
  </p>
  <ol class="ol">
    <li>PM calls <code>spawn_agent</code> with a role name, optional role_id (from the 162-role library),
        a detailed task prompt, and optionally a container image</li>
    <li>The agent CLI sends a POST to <code>/api/projects/{id}/spawn-agent</code></li>
    <li>Backend validates concurrency limits (won&rsquo;t exceed <code>max_concurrent</code>),
        looks up the role system prompt if role_id was provided, creates a launch request with
        project metadata</li>
    <li>Spawner processes the request: resolves workspace path (inheriting from project), constructs
        Docker run arguments with environment variables (<code>PROJECT_ID</code>, <code>PARENT_AGENT_ID</code>,
        <code>ROLE_ID</code>, <code>PROJECT_AGENTS</code> for sibling awareness)</li>
    <li>Container starts, <code>agent-cli</code> registers with the dashboard, fetches its role&rsquo;s
        system prompt from <code>/api/roles/{id}</code>, and begins working on its task</li>
    <li>As the sub-agent works, it streams thinking and tool call events to the dashboard</li>
    <li>When finished, the sub-agent messages the PM via <code>relay_message</code> with a summary
        of findings and created files</li>
    <li>The sub-agent then enters a 5-minute wait loop, listening for follow-up work from the PM</li>
  </ol>

  <div class="h2">3.5 Inter-Agent Messaging</div>
  <p class="p">
    The relay messaging system enables real-time conversations between agents:
  </p>

  <div class="h3">Message Flow</div>
  <ol class="ol">
    <li>Sending agent calls <code>relay_message(target_agent_id, content, wait_for_reply)</code></li>
    <li>Agent CLI POSTs to <code>/api/agents/{id}/relay</code></li>
    <li>Backend inserts message into the <code>messages</code> table for the target agent</li>
    <li>Backend broadcasts <code>message-queued</code> SSE event with agent ID and content</li>
    <li>Target agent&rsquo;s SSE client receives the event, filters by agent ID, pushes content
        to its message channel</li>
    <li>Between agentic loop turns, the target agent&rsquo;s <code>runPrompt()</code> drains the
        message channel and injects messages as <code>[INCOMING MESSAGE FROM AGENT]</code> user messages</li>
    <li>If <code>wait_for_reply=true</code>, the sender blocks for up to 2 minutes waiting for a
        return message</li>
  </ol>

  <div class="h3">Message Injection Between Turns</div>
  <p class="p">
    This is the mechanism that makes the PM interruptible. After every tool execution cycle,
    before the next API call to Claude, the loop performs a non-blocking drain of both the SSE
    channel and the poll endpoint:
  </p>
  <ul class="ul">
    <li>SSE messages are instant (sub-second delivery)</li>
    <li>Poll messages catch anything missed by SSE (10-second fallback)</li>
    <li>Multiple messages are batched into a single user message separated by <code>---</code></li>
    <li>Injected messages appear in the conversation as <code>[INCOMING MESSAGE FROM AGENT]</code>,
        giving Claude the context to respond to sub-agent reports mid-workflow</li>
  </ul>

  <div class="h2">3.6 Session Persistence and Resume</div>
  <p class="p">
    Each agent maintains a persistent session file (<code>{agent-id}.json</code>) containing the
    full system prompt and conversation history (all messages as Anthropic SDK <code>MessageParam</code>
    objects). Sessions are saved after every task completion and on graceful shutdown (SIGINT/SIGTERM).
  </p>
  <p class="p">
    When a project is paused and resumed, the PM agent&rsquo;s container is recreated with the same
    <code>AGENT_ID</code> environment variable. The new container loads the saved session file,
    restoring the full conversation history. This allows the PM to continue exactly where it left
    off, with complete memory of all previous interactions, decisions, and sub-agent results.
  </p>
  <p class="p">
    Resume is handled by the backend&rsquo;s <code>Start</code> endpoint: if a project already has
    a PM agent, it creates a <code>resume</code> launch request instead of a <code>new</code> one,
    passing the existing agent ID. The spawner resolves the correct workspace path from the project
    record and passes it to the new container.
  </p>
</div>

<div class="page">
  <div class="h2">3.7 Sub-Agent Auto-Close Protocol</div>
  <p class="p">
    Sub-agents implement a graceful completion protocol designed to maximise utility while
    preventing resource waste:
  </p>
  <ol class="ol">
    <li>Sub-agent completes its initial task and messages the PM with results</li>
    <li>Enters a 5-minute wait loop, checking both SSE and polling every 5 seconds</li>
    <li>If the PM sends follow-up work (questions, revisions, new tasks), the sub-agent processes
        it and re-enters the wait loop</li>
    <li>If no follow-up arrives within 5 minutes, the agent uploads remaining files, posts a
        completion update with cost summary, calls <code>close_agent</code> on itself, and exits</li>
  </ol>
  <p class="p">
    This design gives the PM a generous window to review output and have multi-round
    conversations with sub-agents before they shut down, while ensuring that idle containers
    don&rsquo;t accumulate indefinitely.
  </p>

  <div class="h2">3.8 Container Spawner Design</div>
  <p class="p">
    The spawner (<code>services/spawner.go</code>) uses a channel-driven architecture for
    zero-latency container launches:
  </p>
  <ul class="ul">
    <li>A buffered channel (<code>chan int64</code>, capacity 100) receives launch request IDs</li>
    <li>A single goroutine processes requests sequentially from the channel</li>
    <li>When a route creates a launch request, it calls <code>spawner.Notify(requestID)</code>,
        which pushes the ID to the channel non-blocking</li>
    <li>The spawner goroutine claims the request (sets status to &ldquo;claimed&rdquo;),
        assembles Docker run arguments, executes <code>docker run -d</code>, and marks
        the request completed or failed</li>
    <li>On startup, any pending requests from before a crash/restart are re-queued</li>
    <li>A separate health-check goroutine polls Docker every 2 minutes for crashed containers</li>
  </ul>
  <p class="p">
    Docker arguments are constructed dynamically based on the launch request metadata, settings
    from the database, and project context. Key mounts include the workspace directory and
    OAuth credentials file (read-only). Resource limits (memory and CPU) are configurable via
    the settings API, defaulting to 2GB RAM and 1 CPU core per container.
  </p>
</div>

<!-- ════════════════════════════════════════════════════════════════════════ -->
<!-- SECTION 4: KEY TECHNICAL DECISIONS                                      -->
<!-- ════════════════════════════════════════════════════════════════════════ -->

<div class="page">
  <div class="h1">4. Key Technical Decisions</div>

  <div class="h2">4.1 Why Go for Backend and Agent CLI</div>
  <p class="p">
    The original prototype was Node.js. The rewrite to Go was motivated by:
  </p>
  <ul class="ul">
    <li><strong>Single binary deployment</strong> &mdash; both the backend server and agent CLI compile
        to single static binaries. The agent CLI is ~8MB and requires no runtime dependencies, making
        the base container image just 49MB (Alpine + bash + curl + git + jq + agent-cli).</li>
    <li><strong>Cross-compilation</strong> &mdash; <code>images/build.sh</code> compiles the agent CLI
        in a <code>golang:1.24-alpine</code> container with <code>-ldflags='-s -w'</code> for minimal
        binary size. The same binary runs in all 7 container images.</li>
    <li><strong>Concurrency model</strong> &mdash; goroutines are ideal for the spawner loop, SSE broker,
        health checker, and backup scheduler. The channel-based spawner design is idiomatic Go.</li>
    <li><strong>Memory efficiency</strong> &mdash; a Node.js agent runtime would add ~80MB per container.
        The Go binary adds ~8MB, enabling the 49MB base image that makes it practical to run
        4-8 simultaneous agents on a laptop.</li>
    <li><strong>Type safety for API</strong> &mdash; the Anthropic Go SDK provides typed message
        construction and response parsing, reducing the risk of malformed API calls.</li>
  </ul>

  <div class="h2">4.2 Why SQLite (Not Postgres)</div>
  <ul class="ul">
    <li><strong>Zero-ops deployment</strong> &mdash; no separate database container. The SQLite file lives
        in a Docker volume (<code>cam-data</code>), persisting across backend rebuilds.</li>
    <li><strong>Single-writer workload</strong> &mdash; agent updates arrive sequentially from HTTP handlers;
        there is no concurrent write contention that would benefit from Postgres.</li>
    <li><strong>WAL mode performance</strong> &mdash; enables concurrent reads (dashboard queries) while
        writes are in progress, sufficient for the single-server deployment model.</li>
    <li><strong>Portable backups</strong> &mdash; the backup scheduler simply copies the database file.
        WAL checkpoint + truncate is called on graceful shutdown.</li>
    <li><strong>Embedded triggers</strong> &mdash; SQLite triggers maintain denormalized counters
        (pending_message_count, unread_update_count) without application-level cache invalidation.</li>
  </ul>

  <div class="h2">4.3 Why Custom Agent CLI (Not Claude CLI npm Package)</div>
  <p class="p">
    Anthropic provides an official Claude CLI as an npm package. The custom agent CLI was built instead for:
  </p>
  <ul class="ul">
    <li><strong>Dashboard integration</strong> &mdash; the CLI needs to register with the backend, stream
        thinking and tool calls, handle relay messages, manage file uploads, and respond to close
        signals. None of this exists in the official CLI.</li>
    <li><strong>PM tool restriction</strong> &mdash; the whitelist enforcement is deeply integrated into the
        agentic loop. PM agents see a filtered tool list and get runtime rejection of blocked tools.</li>
    <li><strong>Inter-turn message injection</strong> &mdash; the ability to drain SSE and poll channels between
        API calls and inject messages into the conversation is fundamental to PM interruptibility.</li>
    <li><strong>Container size</strong> &mdash; the npm CLI requires Node.js (~80MB), plus npm install of the
        package and its dependencies. The Go binary is 8MB with zero dependencies.</li>
    <li><strong>OAuth bridge auth</strong> &mdash; the custom auth flow reads
        <code>.credentials.json</code>, creates a code session, and acquires a bridge worker JWT.
        This works with existing Claude subscriptions without requiring a separate API key.</li>
    <li><strong>Session persistence format</strong> &mdash; sessions are stored as JSON-serialized
        Anthropic SDK <code>MessageParam</code> arrays, enabling exact conversation resumption.</li>
  </ul>
</div>

<div class="page">
  <div class="h2">4.4 Why SSE (Not WebSocket)</div>
  <p class="p">
    Server-Sent Events was chosen over WebSocket for the initial implementation:
  </p>
  <ul class="ul">
    <li><strong>Simpler infrastructure</strong> &mdash; SSE uses standard HTTP, works through all proxies
        and load balancers without special configuration. The SSE broker is ~120 lines of Go.</li>
    <li><strong>Primarily unidirectional</strong> &mdash; the dominant data flow is server-to-client
        (agent updates streaming to dashboard). The rare client-to-server flows (user messages,
        relay requests) are adequately served by REST endpoints.</li>
    <li><strong>Automatic reconnection</strong> &mdash; the browser&rsquo;s <code>EventSource</code> API
        handles reconnection natively. The frontend SSE provider adds a 3-second backoff. The agent
        CLI SSE client implements exponential backoff up to 30 seconds.</li>
    <li><strong>Limitations accepted</strong> &mdash; SSE&rsquo;s lack of bidirectional communication means
        the frontend cannot push messages to agents without a REST call. This is a known limitation
        that WebSocket would address in a future version.</li>
  </ul>

  <div class="h2">4.5 Why Container-Per-Agent (Not Threads/Processes)</div>
  <p class="p">
    Each agent runs in its own Docker container, rather than as a thread or process within the
    backend:
  </p>
  <ul class="ul">
    <li><strong>Tooling isolation</strong> &mdash; a data science agent needs pandas/scipy/matplotlib;
        a web agent needs Playwright/Chromium; a Go agent needs the Go compiler. These cannot
        coexist in a single process without bloating the base image to multi-gigabyte sizes.</li>
    <li><strong>Resource limits</strong> &mdash; Docker provides per-container memory and CPU limits
        (configurable, default 2GB/1 CPU). A runaway agent cannot starve others.</li>
    <li><strong>Crash isolation</strong> &mdash; if an agent crashes (OOM, infinite loop, tool error),
        only that container dies. The backend detects this via the health checker and broadcasts
        a <code>container-health</code> event.</li>
    <li><strong>Security boundary</strong> &mdash; agents run as non-root users with read-only credential
        mounts. The workspace is the only writable mount. Agents cannot access each other&rsquo;s
        containers or the backend&rsquo;s database.</li>
    <li><strong>Clean state</strong> &mdash; each container starts fresh (no process-level state leaks).
        The only persistence is the session file in the workspace and the messages in the database.</li>
  </ul>

  <div class="h2">4.6 OAuth Bridge Discovery for Enterprise Accounts</div>
  <p class="p">
    The authentication system supports two paths:
  </p>
  <ol class="ol">
    <li><strong>Direct API Key</strong> &mdash; set <code>ANTHROPIC_API_KEY</code> environment variable.
        Simplest path, used for API-billed accounts.</li>
    <li><strong>OAuth Bridge</strong> &mdash; reads the OAuth access token from
        <code>~/.claude/.credentials.json</code> (the same file the VS Code extension and official
        Claude CLI write to), creates a code session via <code>POST /v1/code/sessions</code>,
        then acquires a bridge worker via <code>POST /v1/code/sessions/{id}/bridge</code>.
        The bridge response provides a <code>worker_jwt</code> and <code>api_base_url</code>
        that are used with the Anthropic SDK&rsquo;s <code>WithAuthToken</code> and
        <code>WithBaseURL</code> options.</li>
  </ol>
  <p class="p">
    The OAuth bridge tokens expire periodically. The client implements automatic re-authentication:
    when an API call returns 401 or &ldquo;expired token&rdquo;, it calls <code>authenticate()</code>
    again to acquire a fresh bridge session, then retries the request. This is transparent to the
    agentic loop.
  </p>
  <p class="p">
    This design means users with Claude Pro, Team, or Enterprise subscriptions can use Claude
    Projects without purchasing separate API credits. The bridge routes requests through the
    same billing as the official Claude CLI.
  </p>
</div>

<!-- ════════════════════════════════════════════════════════════════════════ -->
<!-- SECTION 5: CURRENT CAPABILITIES                                         -->
<!-- ════════════════════════════════════════════════════════════════════════ -->

<div class="page">
  <div class="h1">5. Current Capabilities</div>

  <div class="h2">5.1 The 162-Role Agent Library</div>
  <p class="p">
    The system ships with a library of 162 pre-defined specialist roles, embedded as a JSON file
    via Go&rsquo;s <code>go:embed</code> directive. Each role contains:
  </p>
  <ul class="ul">
    <li><strong>ID</strong> &mdash; hyphenated identifier (e.g. <code>engineering-backend-architect</code>)</li>
    <li><strong>Name</strong> &mdash; human-readable label (e.g. &ldquo;Backend Architect&rdquo;)</li>
    <li><strong>Category</strong> &mdash; grouping for UI display (Engineering, Testing, Marketing, etc.)</li>
    <li><strong>Description</strong> &mdash; one-line description of the role&rsquo;s expertise</li>
    <li><strong>Vibe</strong> &mdash; personality/communication style descriptor</li>
    <li><strong>Suggested Image</strong> &mdash; recommended container image for this role</li>
    <li><strong>System Prompt</strong> &mdash; multi-paragraph expert system prompt that shapes the
        agent&rsquo;s behaviour, knowledge, and communication style</li>
  </ul>
  <p class="p">
    Image overrides are applied at load time: certain roles are mapped to specialised containers
    (e.g. data roles &rarr; <code>claude-agent-data</code>, web testing roles &rarr;
    <code>claude-agent-web</code>, Go development &rarr; <code>claude-agent-go</code>).
  </p>
  <p class="p">
    The PM agent accesses the library via the <code>search_roles</code> tool, which performs
    keyword matching against role names, descriptions, and categories. This allows the PM to
    discover specialists dynamically based on project needs rather than relying on a fixed set.
  </p>

  <div class="h2">5.2 PM Strategic Decision-Making</div>
  <p class="p">
    The PM agent&rsquo;s effectiveness is shaped by its system prompt, which enforces
    several strategic behaviours:
  </p>
  <ul class="ul">
    <li><strong>Read-React-Adapt</strong> &mdash; after every agent completes, the PM must read the output
        file (via <code>read_file</code>) and make a strategic decision before spawning the next agent.
        This prevents linear plan execution and forces information-driven adaptation.</li>
    <li><strong>Multi-Round Conversations</strong> &mdash; the PM is instructed to have 2-3 exchanges with
        each agent: challenge weak parts, ask for specifics, request rewrites. The
        <code>relay_message</code> tool with <code>wait_for_reply=true</code> enables synchronous
        back-and-forth.</li>
    <li><strong>Timeline Narration</strong> &mdash; the PM posts decisions and reasoning to the project
        timeline via <code>post_update</code>, giving the user visibility into why agents were spawned,
        what was learned, and how the plan is evolving.</li>
    <li><strong>Retry Discipline</strong> &mdash; maximum 3 attempts per task, tracked via scratchpad.
        If a task fails repeatedly, the PM posts an error and moves on rather than wasting resources.</li>
    <li><strong>Completion Verification</strong> &mdash; before marking a project complete, the PM must
        use <code>list_files</code> and <code>read_file</code> to verify all deliverables exist and
        meet quality standards.</li>
  </ul>
</div>

<div class="page">
  <div class="h2">5.3 Real-Time Activity Streaming</div>
  <p class="p">
    Every agent action is streamed to the dashboard in real time via the update posting mechanism:
  </p>
  <ul class="ul">
    <li><strong>Thinking</strong> &mdash; mid-turn text responses (during tool_use stop reasons) are posted
        as <code>thinking</code> updates, truncated to 500 characters. These appear as violet cards
        in the timeline, showing the agent&rsquo;s reasoning process.</li>
    <li><strong>Tool calls</strong> &mdash; every tool invocation is posted as a <code>tool</code> update
        with the tool name as summary and the JSON input as content. The frontend parses the JSON
        to show contextual summaries (file paths, bash commands, search queries, relay targets).</li>
    <li><strong>Completion</strong> &mdash; task completion posts include a structured JSON body with the
        output text (truncated to 5000 chars), list of changed files, token counts, and estimated
        cost in USD.</li>
    <li><strong>Errors</strong> &mdash; API failures, tool errors, and iteration limit exceeded are posted
        as <code>error</code> updates with full error messages.</li>
  </ul>
  <p class="p">
    The rate limiter allows up to 600 updates per minute per agent, ensuring that high-frequency
    tool calling (e.g. a coding agent running many bash commands) doesn&rsquo;t get throttled.
  </p>

  <div class="h2">5.4 PrintingPress PDF Generation</div>
  <p class="p">
    The <code>claude-agent-printingpress</code> image bundles the PrintingPress PDF generation
    system, enabling agents to produce professional branded documents. The PM system prompt
    includes instructions for the PrintingPress API:
  </p>
  <ul class="ul">
    <li>Agent creates a Python script that imports <code>build_document</code> from
        <code>/opt/printingpress/build.py</code></li>
    <li>Content is authored as HTML using PrintingPress CSS component classes
        (<code>.h1</code>, <code>.h2</code>, <code>.p</code>, <code>.table</code>, etc.)</li>
    <li>The script calls <code>build_document(title, subtitle, brand, content_html, output_name, output_dir)</code></li>
    <li>WeasyPrint renders the HTML+CSS to a PDF with branded covers, headers, footers,
        and page numbers</li>
    <li>Available brands: <strong>Reach</strong> (navy corporate) and <strong>Lumi</strong>
        (blue-purple gradient product)</li>
  </ul>

  <div class="h2">5.5 Project Workspace Isolation</div>
  <p class="p">
    Every project gets its own subdirectory within the configured workspace root. When agents
    are spawned, the spawner resolves the host path as
    <code>{workspace_root}/{project_folder_path}</code> and bind-mounts it to
    <code>/workspace</code> in the container. This means:
  </p>
  <ul class="ul">
    <li>All agents in a project share the same filesystem view</li>
    <li>Agents can read each other&rsquo;s output files directly</li>
    <li>Sub-agents cannot access other projects&rsquo; workspaces</li>
    <li>The user can inspect all deliverables in the project folder on the host</li>
    <li>Git operations (if the workspace is a repository) are shared across agents</li>
  </ul>

  <div class="h2">5.6 Token Tracking and Cost Estimation</div>
  <p class="p">
    The agent CLI tracks cumulative input and output tokens using atomic counters. After every
    API call, it logs token counts and estimates cost based on the model&rsquo;s pricing:
  </p>
  <table class="table">
    <tr>
      <th>Model</th>
      <th>Input (per M tokens)</th>
      <th>Output (per M tokens)</th>
    </tr>
    <tr>
      <td>Claude Haiku 4.5</td>
      <td>$1.00</td>
      <td>$5.00</td>
    </tr>
    <tr>
      <td>Claude Sonnet 4.6</td>
      <td>$3.00</td>
      <td>$15.00</td>
    </tr>
    <tr>
      <td>Claude Opus 4.6</td>
      <td>$5.00</td>
      <td>$25.00</td>
    </tr>
  </table>
  <p class="p">
    Cost estimates are included in task completion updates and the auto-close message.
    From test runs, PM agents typically cost $18&ndash;20 per project (dominated by context
    growth over 130+ conversation turns), while sub-agents cost $0.25&ndash;$1 each. A full
    project run producing 45 deliverables costs approximately $25&ndash;30.
  </p>
</div>

<!-- ════════════════════════════════════════════════════════════════════════ -->
<!-- SECTION 6: KNOWN LIMITATIONS & IMPROVEMENT AREAS                        -->
<!-- ════════════════════════════════════════════════════════════════════════ -->

<div class="page">
  <div class="h1">6. Known Limitations and Improvement Areas</div>
  <p class="p">
    This section documents known issues and planned improvements across three dimensions:
    user experience, technical infrastructure, and architectural design.
  </p>

  <div class="h2">6.1 UX/UI Improvements</div>

  <div class="h3">Timeline Ordering</div>
  <p class="p">
    The unified timeline sorts entries by timestamp, but inconsistencies between server-generated
    timestamps (UTC) and client-parsed timestamps can cause ordering anomalies. Updates from
    different agents may interleave in unexpected ways. The timeline should behave as a
    chronological chat-style feed where the most recent activity always appears at the bottom.
  </p>

  <div class="h3">PM Agent Page</div>
  <p class="p">
    The PM agent&rsquo;s detail page currently shows its own thinking and tool calls, but does
    not aggregate the project timeline. Since the PM is the project&rsquo;s orchestrator, its
    page should show a merged view of project-level events, sub-agent status changes, and the
    PM&rsquo;s own activity &mdash; essentially the same view as the project page.
  </p>

  <div class="h3">No WebSocket (SSE Only)</div>
  <p class="p">
    SSE is server-to-client only. User messages sent from the dashboard require a REST call,
    introducing a round-trip that WebSocket would eliminate. More importantly, the dashboard
    cannot push real-time feedback to the PM (e.g. &ldquo;pause this agent&rdquo; or &ldquo;focus
    on the IVF vertical&rdquo;) without the PM polling for messages.
  </p>

  <div class="h3">Page Refresh Requirements</div>
  <p class="p">
    Some state changes (particularly agent linking to projects after spawn) require a page
    refresh to appear in the dashboard. The SSE event system covers most updates, but there are
    gaps in event coverage for certain state transitions.
  </p>

  <div class="h3">No Notification System</div>
  <p class="p">
    When a project completes or an agent requires user input (via <code>ask_user</code>), there
    is no notification mechanism other than the user watching the dashboard. Web Push subscription
    endpoints exist in the API but are not fully integrated into the frontend.
  </p>

  <div class="h3">Mobile Responsiveness</div>
  <p class="p">
    The dashboard has not been tested or optimised for mobile viewports. While Tailwind CSS
    provides responsive utilities, no mobile-specific layouts have been implemented.
  </p>
</div>

<div class="page">
  <div class="h2">6.2 Technical Improvements</div>

  <div class="h3">PM Conversation Quality</div>
  <p class="p">
    The most significant functional limitation. Despite the system prompt&rsquo;s instructions,
    the PM agent tends toward linear plan execution rather than adaptive strategy. Symptoms:
  </p>
  <ul class="ul">
    <li>PM builds a rigid plan at project start and executes it sequentially</li>
    <li>Agent findings rarely change the PM&rsquo;s subsequent decisions</li>
    <li>Multi-round conversations (PM asking follow-up questions) happen inconsistently</li>
    <li>Output tends toward volume (&ldquo;554 pages of sophisticated filler&rdquo;) rather than
        focused quality (&ldquo;15 pages of killer strategy&rdquo;)</li>
  </ul>
  <p class="p">
    This is fundamentally a prompt engineering and conversation architecture challenge. The
    message injection mechanism works &mdash; messages arrive between turns &mdash; but the PM
    does not consistently react to them with strategic depth. Potential mitigations include
    structured phase gates, mandatory read-then-decide checkpoints, and example conversations
    in the system prompt.
  </p>

  <div class="h3">Sub-Agent Completion Protocol</div>
  <p class="p">
    The auto-close timer (5 minutes) creates a tension: long enough for the PM to review and
    respond, but sometimes the PM is mid-turn when a sub-agent times out. The PM then tries to
    message a closed agent, gets an error, and may retry unnecessarily. A more robust protocol
    would have the PM explicitly acknowledge receipt of a sub-agent&rsquo;s report, resetting the
    timeout.
  </p>

  <div class="h3">No Automated Testing</div>
  <p class="p">
    The codebase has no unit tests, integration tests, or end-to-end tests. Given the complexity
    of the spawner, relay messaging, session persistence, and PM tool restriction logic, this is
    a significant risk for regressions during development. The monolithic agent CLI is particularly
    difficult to test without dependency injection.
  </p>

  <div class="h3">No CI/CD Pipeline</div>
  <p class="p">
    There is no automated build, test, or deployment pipeline. Agent images must be rebuilt
    manually via <code>images/build.sh</code> after any change to the agent CLI. The backend
    and frontend are built by Docker Compose on <code>docker compose up --build</code>.
  </p>

  <div class="h3">Manual Image Rebuilds</div>
  <p class="p">
    After modifying any file in <code>backend/cmd/agent-cli/</code>, all 7 agent images must be
    rebuilt because they share the same compiled binary. The build script handles this
    (cross-compile once, copy to all), but it must be run manually.
  </p>

  <div class="h3">Database Migration Limitations</div>
  <p class="p">
    Migrations are additive-only (ADD COLUMN). There is no version tracking, no rollback
    capability, and no support for column removal or type changes. The schema table recreation
    (for removing CHECK constraints) is a one-off workaround, not a generalised migration system.
  </p>

  <div class="h3">No Rate Limiting for API Costs</div>
  <p class="p">
    There is no per-project or per-agent budget limit. A PM agent with a concurrency limit of 4
    could spawn agents indefinitely (closing old ones to free slots), potentially running up
    significant Anthropic API costs. Token counters exist in the agent CLI but are not persisted
    to the database or aggregated at the project level.
  </p>

  <div class="h3">Token Counter Persistence</div>
  <p class="p">
    Token counts use in-memory atomic counters that are lost when the container exits. The
    completion update includes the final token count, but there is no cumulative project-level
    cost tracking in the database.
  </p>
</div>

<div class="page">
  <div class="h2">6.3 Architectural Improvements</div>

  <div class="h3">Domain Service Layer</div>
  <p class="p">
    HTTP handlers currently contain business logic (e.g. <code>projects.go Start()</code>
    generates PM prompts, creates launch requests, and manages project state transitions).
    A dedicated service layer would separate HTTP concerns from business rules, enabling
    reuse (e.g. starting a project from a workflow or API without going through HTTP) and
    making the code testable without HTTP mocking.
  </p>

  <div class="h3">No Dependency Injection</div>
  <p class="p">
    Services are instantiated directly in <code>main.go</code> and passed to route constructors.
    The agent CLI uses package-level globals (<code>activeDashboard</code>,
    <code>activeSSE</code>). This makes unit testing impractical &mdash; you cannot inject
    mock implementations of the SSE client, dashboard client, or Anthropic client.
  </p>

  <div class="h3">Agent CLI Monolith</div>
  <p class="p">
    The agent CLI is a single Go package (<code>cmd/agent-cli/</code>) where the agentic loop,
    tool definitions, tool implementations, session management, dashboard client, SSE client,
    and authentication are all in the same package. Splitting into separate packages (e.g.
    <code>loop</code>, <code>tools</code>, <code>comms</code>, <code>auth</code>) would improve
    testability and readability.
  </p>

  <div class="h3">WebSocket for Bidirectional Communication</div>
  <p class="p">
    WebSocket would enable:
  </p>
  <ul class="ul">
    <li>Real-time user messages to agents without REST round-trips</li>
    <li>Streaming token-by-token output (currently only post-turn summaries)</li>
    <li>Bidirectional health pings between agents and backend</li>
    <li>Push-based agent lifecycle commands (pause, resume, cancel) without polling</li>
  </ul>

  <div class="h3">gRPC Between Backend and Agents</div>
  <p class="p">
    The current REST+SSE communication between agents and the backend could be replaced with
    gRPC for:
  </p>
  <ul class="ul">
    <li>Typed message contracts (protobuf) instead of ad-hoc JSON</li>
    <li>Bidirectional streaming for real-time activity feeds</li>
    <li>Multiplexed connections (single TCP connection per agent)</li>
    <li>Automatic retry and deadline propagation</li>
  </ul>
  <p class="p">
    However, gRPC would add complexity to the Docker networking setup (agents currently use
    <code>host.docker.internal</code> for HTTP, which is simpler than configuring gRPC service
    discovery) and would not provide immediate user-facing benefits.
  </p>
</div>

<!-- ════════════════════════════════════════════════════════════════════════ -->
<!-- SECTION 7: DEPLOYMENT AND OPERATIONS                                    -->
<!-- ════════════════════════════════════════════════════════════════════════ -->

<div class="page">
  <div class="h1">7. Deployment and Operations</div>

  <div class="h2">7.1 Docker Compose Configuration</div>
  <p class="p">
    The system deploys via a minimal <code>docker-compose.yml</code> with two services:
  </p>
  <table class="table">
    <tr>
      <th style="width:15%">Service</th>
      <th style="width:12%">Port</th>
      <th style="width:73%">Configuration</th>
    </tr>
    <tr>
      <td><code>cam</code></td>
      <td>9222</td>
      <td>Backend Go server. Mounts: <code>cam-data</code> volume for SQLite, Docker socket for
          container spawning, workspace root (read-only for file browsing). Restart policy:
          <code>unless-stopped</code>.</td>
    </tr>
    <tr>
      <td><code>frontend</code></td>
      <td>4173</td>
      <td>Next.js dashboard. Environment: <code>NEXT_PUBLIC_API_URL=http://localhost:9222</code>.
          Depends on <code>cam</code> service. Restart policy: <code>unless-stopped</code>.</td>
    </tr>
  </table>
  <p class="p">
    The backend container has Docker socket access (<code>/var/run/docker.sock</code>), which
    allows it to spawn and manage agent containers. This is the mechanism that gives the backend
    control over the agent fleet without requiring a separate orchestrator (Kubernetes, Docker
    Swarm, etc.).
  </p>

  <div class="h2">7.2 Setup Wizard</div>
  <p class="p">
    On first launch, the frontend detects that setup is incomplete (via
    <code>GET /api/settings/setup-status</code>) and presents a setup wizard that configures:
  </p>
  <ul class="ul">
    <li><strong>Claude Config Path</strong> &mdash; path to the directory containing
        <code>.claude/.credentials.json</code> on the host, used for OAuth auth mount</li>
    <li><strong>Workspace Root</strong> &mdash; host directory where project folders are created</li>
    <li><strong>Agent Memory Limit</strong> &mdash; per-container RAM limit (default: 2GB)</li>
    <li><strong>Agent CPU Limit</strong> &mdash; per-container CPU limit (default: 1 core)</li>
    <li><strong>Max Concurrent Agents</strong> &mdash; global limit (default: 8)</li>
    <li><strong>Default Agent Image</strong> &mdash; default container image (default: <code>claude-agent</code>)</li>
  </ul>

  <div class="h2">7.3 Build Process</div>
  <p class="p">
    A full deployment involves three build steps:
  </p>
  <ol class="ol">
    <li><strong>Agent images</strong> &mdash; <code>bash images/build.sh</code> cross-compiles the
        agent CLI binary in a Go container, then builds all 7 Docker images. Each image copies the
        pre-compiled binary and adds its specific dependencies.</li>
    <li><strong>Backend + Frontend</strong> &mdash; <code>docker compose up -d --build</code> builds the
        backend Go server (multi-stage Dockerfile) and the Next.js frontend, then starts both services.</li>
    <li><strong>Setup</strong> &mdash; first visit to <code>http://localhost:4173</code> triggers the
        setup wizard for credential paths and workspace configuration.</li>
  </ol>

  <div class="h2">7.4 Graceful Shutdown</div>
  <p class="p">
    The backend implements graceful shutdown on SIGINT/SIGTERM:
  </p>
  <ol class="ol">
    <li>Broadcasts a <code>shutdown</code> SSE event so agents know to reconnect, not exit</li>
    <li>HTTP server shutdown with 20-second timeout for in-flight requests</li>
    <li>SQLite WAL checkpoint and truncate to ensure database consistency</li>
    <li>Database connection close</li>
  </ol>
  <p class="p">
    Agent containers survive backend restarts because they reconnect via SSE with exponential
    backoff. The SSE client treats a <code>shutdown</code> event as a signal to reconnect rather
    than terminate, maintaining agent session continuity across backend deployments.
  </p>
</div>

<!-- ════════════════════════════════════════════════════════════════════════ -->
<!-- SECTION 8: TEST RESULTS AND PRODUCTION EXPERIENCE                       -->
<!-- ════════════════════════════════════════════════════════════════════════ -->

<div class="page">
  <div class="h1">8. Test Results and Production Experience</div>

  <div class="h2">8.1 ICP Campaign Test Run</div>
  <p class="p">
    The first full production test was a Lumi ICP (Ideal Customer Profile) campaign project.
    Results:
  </p>
  <table class="table">
    <tr>
      <th style="width:30%">Metric</th>
      <th style="width:70%">Result</th>
    </tr>
    <tr>
      <td>Total agents spawned</td>
      <td>~15 specialist agents across multiple phases</td>
    </tr>
    <tr>
      <td>Files produced</td>
      <td>45 files, 960KB total</td>
    </tr>
    <tr>
      <td>Run time</td>
      <td>~40 minutes</td>
    </tr>
    <tr>
      <td>PM cost</td>
      <td>~$18&ndash;20 (130+ conversation turns)</td>
    </tr>
    <tr>
      <td>Sub-agent cost (each)</td>
      <td>$0.25&ndash;$1</td>
    </tr>
    <tr>
      <td>Total run cost</td>
      <td>~$25&ndash;30</td>
    </tr>
  </table>

  <div class="h3">What Worked</div>
  <ul class="ul">
    <li>162-role library loaded and served correctly via API</li>
    <li>Role picker UI functional in the dashboard</li>
    <li>Agents successfully writing files to the shared workspace</li>
    <li><code>max_tokens</code> continuation (agents no longer stop mid-write)</li>
    <li>Sub-agent auto-close with file uploads</li>
    <li>Workspace file browser accessible from the dashboard</li>
    <li>PM resume without creating duplicate agents</li>
    <li>Archived agents collapsed in UI to reduce clutter</li>
    <li>Timeline timestamps displaying correctly</li>
  </ul>

  <div class="h3">Issues Discovered</div>
  <ul class="ul">
    <li>PM executed a linear plan without reacting to agent findings</li>
    <li>PM did not have genuine back-and-forth conversations with agents</li>
    <li>Output was high-volume but low-quality (554 pages of broad coverage vs 15 pages of focused strategy)</li>
    <li>Some agents overwrote sibling agents&rsquo; output files</li>
    <li>Agent-to-project linking failed for later-spawned agents (timing issue, since fixed)</li>
    <li>PM could not see agents via <code>list_project_agents</code> due to linking failure</li>
  </ul>

  <div class="h2">8.2 Code Review Findings</div>
  <p class="p">
    A comprehensive code review of the codebase identified:
  </p>
  <table class="table">
    <tr>
      <th>Component</th>
      <th>Critical</th>
      <th>High</th>
      <th>Total</th>
    </tr>
    <tr>
      <td>Backend</td>
      <td class="td-red">4</td>
      <td class="td-amber">10</td>
      <td>43</td>
    </tr>
    <tr>
      <td>Frontend</td>
      <td class="td-red">3</td>
      <td class="td-amber">5</td>
      <td>28</td>
    </tr>
  </table>

  <div class="h3">Critical Backend Issues</div>
  <ul class="ul">
    <li>SQL injection in <code>GetDistinctAgentIDs</code> (user input concatenated into query)</li>
    <li>Nil pointer panic in <code>relay_message</code> when dashboard client is nil</li>
    <li>Nil pointer panic in <code>tools_schedule</code></li>
    <li>Race condition on atomic token counters (read-then-update not atomic)</li>
  </ul>

  <div class="h3">Critical Frontend Issues</div>
  <ul class="ul">
    <li>XSS vulnerability via <code>dangerouslySetInnerHTML</code> in the timeline
        (since mitigated with HTML entity escaping)</li>
    <li>API key exposed in SSE URL query parameter (visible in browser dev tools and server logs)</li>
    <li>Unauthenticated key endpoint (<code>GET /api/auth/key</code> returns the API key without auth)</li>
  </ul>
</div>

<div class="page">
  <div class="h2">8.3 Fixes Applied Across Test Sessions</div>
  <p class="p">
    Multiple test-fix-test cycles addressed the most impactful issues:
  </p>

  <div class="h3">Session 1 Fixes</div>
  <ul class="ul">
    <li>System prompt separation: PM prompt as system, user description as first user message</li>
    <li>Agent title propagation from role name instead of generic &ldquo;Container Agent&rdquo;</li>
    <li>File upload on task completion (auto-upload changed files)</li>
    <li>Workspace file browser working after mount path resolution fix</li>
  </ul>

  <div class="h3">Session 2 Fixes</div>
  <ul class="ul">
    <li>Tool and thinking streaming to timeline (CHECK constraint on <code>updates.type</code>
        was blocking new types &mdash; table recreated without constraint)</li>
    <li>PM restricted to orchestration-only tools (13-tool whitelist with double enforcement)</li>
    <li>Sub-agent auto-close after 5-minute idle window</li>
    <li>SSE reconnect on dashboard restart (agents survive via reconnect loop)</li>
    <li>Agent linking fixed (project_id, role, parent_agent_id sent as env vars instead of
        fragile launch request matching)</li>
    <li>New Next.js frontend with unified timeline, auth gate, SSE provider</li>
    <li>Rate limiter increased to 600/min for streaming-heavy agents</li>
    <li>XSS mitigation via HTML entity escaping in <code>FormattedText</code> component</li>
    <li>Continuous conversation mode for PM (messages feed into same session, not isolated tasks)</li>
    <li>Inter-turn message injection (SSE + poll drain between API calls)</li>
  </ul>

  <div class="h2">8.4 The Fundamental Challenge</div>
  <p class="p">
    The core remaining challenge is not infrastructure but intelligence. The system reliably
    spawns agents, delivers messages, streams activity, and manages lifecycles. But the PM
    agent&rsquo;s strategic behaviour &mdash; reading output, adapting plans, making hard calls,
    and having genuine multi-round conversations &mdash; is inconsistent.
  </p>
  <p class="p">
    This is a prompt engineering challenge at the frontier of what current models can do
    reliably. The PM needs to:
  </p>
  <ol class="ol">
    <li><strong>Read and react</strong> &mdash; change plans based on what agents discover</li>
    <li><strong>Make hard calls</strong> &mdash; choose 1-2 directions instead of covering everything</li>
    <li><strong>Have real conversations</strong> &mdash; reject weak work, request specifics, iterate</li>
    <li><strong>Build on findings</strong> &mdash; later agents should use earlier agents&rsquo; discoveries</li>
    <li><strong>Cut ruthlessly</strong> &mdash; final output should be focused and actionable, not comprehensive</li>
  </ol>
  <p class="p">
    The test for success: does the PM change its plan based on what agents discover? If yes,
    the system is working as designed. If no, the orchestration is still mechanical rather
    than intelligent.
  </p>
</div>

<!-- ════════════════════════════════════════════════════════════════════════ -->
<!-- SECTION 9: FUTURE ROADMAP                                               -->
<!-- ════════════════════════════════════════════════════════════════════════ -->

<div class="page">
  <div class="h1">9. Future Roadmap</div>

  <div class="h2">9.1 V2 Architecture</div>
  <p class="p">
    The following architectural changes are planned for the next major version:
  </p>

  <div class="h3">WebSocket Integration</div>
  <p class="p">
    Replace SSE with WebSocket for bidirectional real-time communication. This enables:
    real-time user messages to agents, streaming token-by-token output, push-based lifecycle
    commands, and bidirectional health monitoring. The SSE provider in the frontend would be
    replaced with a WebSocket provider maintaining a single multiplexed connection.
  </p>

  <div class="h3">Interruptible PM Loop</div>
  <p class="p">
    The PM&rsquo;s agentic loop should be concurrent with message handling at the model level,
    not just at the turn boundary. Sub-agent completion messages should be able to interrupt the
    PM&rsquo;s current API call (by injecting into a pending request&rsquo;s context), not wait
    for the current turn to complete. This would make conversations more natural and reduce
    the delay between sub-agent completion and PM reaction.
  </p>

  <div class="h3">Service Layer Extraction</div>
  <p class="p">
    Extract business logic from HTTP handlers into a domain service layer. The project start
    flow (PM prompt generation, launch request creation, project state transition) should be a
    single service method callable from HTTP handlers, workflow engine, and future gRPC endpoints.
  </p>

  <div class="h3">Dependency Injection</div>
  <p class="p">
    Introduce interfaces for all external dependencies (Anthropic client, Docker client,
    database, SSE broker) and constructor-based injection. This enables unit testing with
    mock implementations and makes the architecture explicit about its dependencies.
  </p>

  <div class="h2">9.2 Operational Improvements</div>
  <ul class="ul">
    <li><strong>Cost tracking dashboard</strong> &mdash; per-project and per-agent token and cost
        aggregation, persisted to database, visible in the frontend</li>
    <li><strong>Budget limits</strong> &mdash; per-project cost caps that pause the PM when exceeded</li>
    <li><strong>Automated testing</strong> &mdash; unit tests for spawner, relay, session persistence;
        integration tests for the full spawn-work-report cycle using mock Anthropic responses</li>
    <li><strong>CI/CD pipeline</strong> &mdash; automated image builds and backend deployment on push</li>
    <li><strong>Database migration system</strong> &mdash; versioned, reversible migrations with
        rollback support</li>
    <li><strong>Notification system</strong> &mdash; Web Push integration for project completion,
        agent questions, and error alerts</li>
  </ul>

  <div class="h2">9.3 PM Intelligence</div>
  <p class="p">
    The highest-impact improvement area. Approaches under consideration:
  </p>
  <ul class="ul">
    <li><strong>Structured phase gates</strong> &mdash; force the PM to post a
        &ldquo;Phase N Decision&rdquo; update with explicit reasoning about what was learned and
        how the plan is changing before spawning the next batch of agents</li>
    <li><strong>Mandatory output review</strong> &mdash; inject a system-level constraint that the PM
        must call <code>read_file</code> on every agent&rsquo;s output before spawning any new agent</li>
    <li><strong>Example conversations</strong> &mdash; include multi-round PM-agent conversation examples
        in the system prompt to demonstrate the expected depth of interaction</li>
    <li><strong>Output quality scoring</strong> &mdash; a lightweight self-evaluation step where the PM
        rates each agent&rsquo;s output on a scale and explains the rating before deciding next steps</li>
    <li><strong>Smaller context window</strong> &mdash; aggressively summarise old conversation turns to
        keep the PM&rsquo;s context focused on recent findings rather than the full history</li>
  </ul>
</div>

<!-- ════════════════════════════════════════════════════════════════════════ -->
<!-- SECTION 10: APPENDIX                                                    -->
<!-- ════════════════════════════════════════════════════════════════════════ -->

<div class="page">
  <div class="h1">Appendix A: Complete API Endpoint Reference</div>
  <p class="p">
    All endpoints are prefixed with <code>/api/</code> and require API key authentication
    (except health checks and setup status).
  </p>

  <table class="table">
    <tr>
      <th style="width:12%">Method</th>
      <th style="width:40%">Path</th>
      <th style="width:48%">Description</th>
    </tr>
    <tr><td>GET</td><td>/health</td><td>Server health check</td></tr>
    <tr><td>GET</td><td>/health/db</td><td>Database integrity check</td></tr>
    <tr><td>GET</td><td>/events</td><td>SSE event stream</td></tr>
    <tr><td>GET</td><td>/agents</td><td>List all agents</td></tr>
    <tr><td>GET</td><td>/agents/analytics</td><td>Agent statistics</td></tr>
    <tr><td>GET</td><td>/agents/{id}</td><td>Get agent details</td></tr>
    <tr><td>PATCH</td><td>/agents/{id}</td><td>Update agent fields</td></tr>
    <tr><td>DELETE</td><td>/agents/{id}</td><td>Delete agent</td></tr>
    <tr><td>POST</td><td>/agents/{id}/updates</td><td>Post activity update</td></tr>
    <tr><td>GET</td><td>/agents/{id}/updates</td><td>Get agent timeline</td></tr>
    <tr><td>POST</td><td>/agents/{id}/messages</td><td>Send message to agent</td></tr>
    <tr><td>GET</td><td>/agents/{id}/messages</td><td>Get agent messages</td></tr>
    <tr><td>POST</td><td>/agents/{id}/close</td><td>Archive and close agent</td></tr>
    <tr><td>POST</td><td>/agents/{id}/resume</td><td>Resume archived agent</td></tr>
    <tr><td>POST</td><td>/agents/{id}/relay</td><td>Relay message to another agent</td></tr>
    <tr><td>POST</td><td>/agents/{id}/files</td><td>Upload file</td></tr>
    <tr><td>GET</td><td>/agents/{id}/files</td><td>List agent files</td></tr>
    <tr><td>GET</td><td>/agents/{id}/files/{fid}</td><td>Download file</td></tr>
    <tr><td>GET</td><td>/projects</td><td>List all projects</td></tr>
    <tr><td>POST</td><td>/projects</td><td>Create project</td></tr>
    <tr><td>GET</td><td>/projects/{id}</td><td>Get project details</td></tr>
    <tr><td>PATCH</td><td>/projects/{id}</td><td>Update project</td></tr>
    <tr><td>DELETE</td><td>/projects/{id}</td><td>Delete project</td></tr>
    <tr><td>POST</td><td>/projects/{id}/start</td><td>Start or resume project</td></tr>
    <tr><td>POST</td><td>/projects/{id}/pause</td><td>Pause project</td></tr>
    <tr><td>POST</td><td>/projects/{id}/complete</td><td>Complete project</td></tr>
    <tr><td>GET</td><td>/projects/{id}/agents</td><td>Get project agents</td></tr>
    <tr><td>POST</td><td>/projects/{id}/spawn-agent</td><td>Spawn agent in project</td></tr>
    <tr><td>GET</td><td>/projects/{id}/unified-timeline</td><td>Merged timeline</td></tr>
    <tr><td>GET</td><td>/roles</td><td>List all roles (162)</td></tr>
    <tr><td>GET</td><td>/roles/stats</td><td>Role library statistics</td></tr>
    <tr><td>GET</td><td>/roles/{id}</td><td>Get role with system prompt</td></tr>
    <tr><td>GET</td><td>/settings</td><td>Get all settings</td></tr>
    <tr><td>PATCH</td><td>/settings</td><td>Update settings</td></tr>
  </table>
</div>

<div class="page">
  <div class="h1">Appendix B: Environment Variables</div>
  <p class="p">
    Environment variables used by the backend server and agent CLI.
  </p>

  <div class="h2">Backend Server</div>
  <table class="table">
    <tr>
      <th style="width:30%">Variable</th>
      <th style="width:15%">Default</th>
      <th style="width:55%">Description</th>
    </tr>
    <tr><td><code>PORT</code></td><td>9222</td><td>HTTP server port</td></tr>
    <tr><td><code>DB_PATH</code></td><td>./data/agents.db</td><td>SQLite database file path</td></tr>
    <tr><td><code>HOST_HOME_MOUNT</code></td><td>/host-home</td><td>Mounted host home directory for folder browsing</td></tr>
  </table>

  <div class="h2">Agent CLI (set by spawner)</div>
  <table class="table">
    <tr>
      <th style="width:30%">Variable</th>
      <th style="width:70%">Description</th>
    </tr>
    <tr><td><code>AGENT_URL</code></td><td>Dashboard API base URL (e.g. http://host.docker.internal:9222). Triggers agent mode when set.</td></tr>
    <tr><td><code>AGENT_ID</code></td><td>Agent UUID (set for resumed agents to load correct session)</td></tr>
    <tr><td><code>AGENT_TITLE</code></td><td>Display name for the agent (role name or &ldquo;Agent&rdquo;)</td></tr>
    <tr><td><code>DASHBOARD_API_KEY</code></td><td>API key for authenticating with the backend</td></tr>
    <tr><td><code>PROJECT_ID</code></td><td>UUID of the parent project (enables orchestration tools)</td></tr>
    <tr><td><code>PARENT_AGENT_ID</code></td><td>UUID of the PM agent (set for sub-agents)</td></tr>
    <tr><td><code>ROLE_ID</code></td><td>Role library ID for fetching specialized system prompt</td></tr>
    <tr><td><code>PROJECT_AGENTS</code></td><td>Pipe-separated list of sibling agents (id:role:status)</td></tr>
    <tr><td><code>ANTHROPIC_API_KEY</code></td><td>Direct API key (bypasses OAuth bridge)</td></tr>
    <tr><td><code>MODEL</code></td><td>Model ID (default: claude-sonnet-4-20250514)</td></tr>
    <tr><td><code>THINKING</code></td><td>Set to &ldquo;true&rdquo; to enable extended thinking (10K token budget)</td></tr>
  </table>
</div>

<div class="page">
  <div class="h1">Appendix C: Database Schema Diagram</div>
  <p class="p">
    Logical relationships between the 10 database tables.
  </p>

  <table class="table">
    <tr>
      <th style="width:25%">Relationship</th>
      <th style="width:75%">Description</th>
    </tr>
    <tr>
      <td><code>projects</code> &rarr; <code>agents</code></td>
      <td>One-to-many via <code>agents.project_id</code>. A project has one PM agent and
          zero or more sub-agents.</td>
    </tr>
    <tr>
      <td><code>agents</code> &rarr; <code>agents</code></td>
      <td>Self-referential via <code>agents.parent_agent_id</code>. Sub-agents reference
          their PM agent.</td>
    </tr>
    <tr>
      <td><code>agents</code> &rarr; <code>updates</code></td>
      <td>One-to-many via <code>updates.agent_id</code>. Agent activity timeline entries.</td>
    </tr>
    <tr>
      <td><code>agents</code> &rarr; <code>messages</code></td>
      <td>One-to-many via <code>messages.agent_id</code>. Incoming messages for an agent,
          with optional <code>source_agent_id</code> for inter-agent relay.</td>
    </tr>
    <tr>
      <td><code>agents</code> &rarr; <code>files</code></td>
      <td>One-to-many via <code>files.agent_id</code>. Uploaded file metadata (data stored on disk).</td>
    </tr>
    <tr>
      <td><code>projects</code> &rarr; <code>project_updates</code></td>
      <td>One-to-many via <code>project_updates.project_id</code>. Project-level timeline events.</td>
    </tr>
    <tr>
      <td><code>launch_requests</code></td>
      <td>Standalone queue table. Links to agents via metadata JSON in <code>agent_id</code> field
          (stores project_id, role, parent_agent_id, prompt as JSON blob).</td>
    </tr>
    <tr>
      <td><code>settings</code></td>
      <td>Key-value store. No foreign keys. Stores workspace_root, claude_config_path,
          resource limits, API key, and setup state.</td>
    </tr>
    <tr>
      <td><code>workflows</code></td>
      <td>Standalone. Multi-step workflow definitions with JSON step arrays.</td>
    </tr>
    <tr>
      <td><code>webhooks</code></td>
      <td>Standalone. External notification targets with failure tracking.</td>
    </tr>
  </table>

  <div class="h2">Index Strategy</div>
  <p class="p">
    The schema defines 8 indexes optimised for the most common query patterns:
  </p>
  <ul class="ul">
    <li><code>idx_agents_project</code> &mdash; list agents by project (project detail page)</li>
    <li><code>idx_agents_status</code> &mdash; filter agents by status (dashboard list)</li>
    <li><code>idx_updates_agent_id</code> and <code>idx_updates_agent</code> &mdash; agent timeline queries</li>
    <li><code>idx_messages_agent_id</code> and <code>idx_messages_agent_status</code> &mdash; pending message delivery</li>
    <li><code>idx_files_agent_id</code> &mdash; agent file listings</li>
    <li><code>idx_project_updates_project_id</code> &mdash; project timeline queries</li>
    <li><code>idx_launch_requests_status</code> &mdash; spawner queue processing</li>
  </ul>
</div>

"""

# ─── Build the PDF ───────────────────────────────────────────────────────────

build_document(
    title='Claude Agent Manager',
    subtitle='Technical Architecture Report',
    brand='reach',
    layout='portrait',
    classification='internal',
    cover_eyebrow='Technical Report &middot; April 2026',
    cover_pills=['Go Backend', 'Next.js Frontend', 'Docker Agents'],
    cover_bottom_bar=['Reach Industries', 'Claude Projects v1.0'],
    content_html=CONTENT,
    output_name='CAM_Technical_Report',
    output_dir=os.path.join(os.path.dirname(os.path.abspath(__file__)), 'output'),
    date_str='April 2026',
)
