"""
Claude Projects — Technical Comparison Report
Old (ClaudeAgentManager v3.0.0) vs New (Claude Projects)

Build with: ~/Desktop/PrintingPress/build.sh docs/technical_report.py
"""
import sys, os
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from build import build_document

CONTENT = """

<div class="page">
  <div class="h1">Executive Summary</div>

  <div class="intro-card">
    Claude Projects is a ground-up rewrite of ClaudeAgentManager v3.0.0, replacing the Node.js backend and Windows Terminal launcher with a Go backend and Docker container-based agent system.
  </div>

  <p class="p">The original ClaudeAgentManager was a Node.js application designed for a single Windows workstation. Agents ran as native terminal processes with full host filesystem access. The new system runs everything in Docker containers with a custom Go CLI providing 26 built-in tools and native inter-agent communication.</p>

  <div class="expect">
    <span class="expect-label">Key Result</span>
    <p class="p" style="margin-bottom:0;">Agent image size: 311MB &rarr; 49MB (84% reduction). Host dependencies: Node.js + Claude CLI &rarr; Docker only. Agent isolation: none &rarr; full container sandboxing.</p>
  </div>
</div>

<div class="page">
  <div class="h1">Architecture Comparison</div>

  <table class="table">
    <tr><th>Aspect</th><th>Old (v3.0.0)</th><th>New (Claude Projects)</th></tr>
    <tr><td><strong>Backend</strong></td><td>Node.js Express + TypeScript</td><td>Go 1.24 (stdlib only)</td></tr>
    <tr><td><strong>Frontend</strong></td><td>Separate Nginx container</td><td>Embedded in Go binary</td></tr>
    <tr><td><strong>Dashboard image</strong></td><td>~400MB</td><td>32MB</td></tr>
    <tr><td><strong>Agent image</strong></td><td class="td-red">311MB (Node.js + npm CLI)</td><td class="td-green">49MB (Alpine + Go binary)</td></tr>
    <tr><td><strong>Agent runtime</strong></td><td class="td-red">npm Claude CLI + bash wrapper</td><td class="td-green">Custom Go binary, 26 tools</td></tr>
    <tr><td><strong>Agent execution</strong></td><td class="td-red">Windows Terminal (wt.exe)</td><td class="td-green">Docker containers</td></tr>
    <tr><td><strong>Host dependencies</strong></td><td class="td-red">Node.js + Claude CLI</td><td class="td-green">Docker only</td></tr>
    <tr><td><strong>Agent isolation</strong></td><td class="td-red">None &mdash; full host access</td><td class="td-green">Container sandbox</td></tr>
    <tr><td><strong>Message delivery</strong></td><td class="td-amber">3-second polling</td><td class="td-green">SSE (instant)</td></tr>
    <tr><td><strong>Inter-agent comms</strong></td><td class="td-amber">bash + curl</td><td class="td-green">Native tool</td></tr>
    <tr><td><strong>Session persistence</strong></td><td class="td-red">--continue flag (fragile)</td><td class="td-green">JSON session files</td></tr>
    <tr><td><strong>Configuration</strong></td><td class="td-amber">Env vars + hardcoded paths</td><td class="td-green">Browser setup wizard</td></tr>
    <tr><td><strong>OS support</strong></td><td class="td-red">Windows only</td><td class="td-green">Any (Docker)</td></tr>
  </table>
</div>

<div class="page">
  <div class="h1">Custom Go CLI: 26 Tools</div>

  <table class="table">
    <tr><th>Category</th><th>Tools</th></tr>
    <tr><td>Execution</td><td>bash</td></tr>
    <tr><td>Files</td><td>read_file (line ranges), write_file, edit_file (replace_all), list_files</td></tr>
    <tr><td>Search</td><td>grep (regex), glob (recursive)</td></tr>
    <tr><td>Web</td><td>web_fetch (HTML-to-text), web_search</td></tr>
    <tr><td>Agent Orchestration</td><td>spawn_agent, relay_message, list_project_agents, post_update</td></tr>
    <tr><td>Task Management</td><td>task_create, task_update, task_list</td></tr>
    <tr><td>Interactive</td><td>ask_user (SSE-based), plan_mode</td></tr>
    <tr><td>Scheduling</td><td>schedule_task, cancel_schedule, list_schedules</td></tr>
    <tr><td>Code Intelligence</td><td>find_definition, find_references</td></tr>
    <tr><td>Notebooks</td><td>notebook_edit</td></tr>
  </table>

  <p class="p">The npm Claude CLI provides tools determined by Anthropic. Our Go CLI gives full control: tools are purpose-built for agent orchestration, dashboard integration is native (not bash wrappers), and session persistence is built into the binary.</p>
</div>

<div class="page">
  <div class="h1">Metrics</div>

  <table class="table">
    <tr><th>Metric</th><th>Old</th><th>New</th><th>Change</th></tr>
    <tr><td>Dashboard image</td><td>~400MB</td><td>32MB</td><td class="td-green">&minus;92%</td></tr>
    <tr><td>Agent image</td><td>311MB</td><td>49MB</td><td class="td-green">&minus;84%</td></tr>
    <tr><td>Docker containers</td><td>2 (backend + nginx)</td><td>1</td><td class="td-green">&minus;50%</td></tr>
    <tr><td>Host dependencies</td><td>Node.js + Claude CLI</td><td>Docker only</td><td class="td-green">Eliminated</td></tr>
    <tr><td>Message latency</td><td>~3s (poll)</td><td>&lt;100ms (SSE)</td><td class="td-green">&minus;97%</td></tr>
    <tr><td>Agent tools</td><td>Claude CLI defaults</td><td>26 custom</td><td class="td-green">Full control</td></tr>
    <tr><td>OS support</td><td>Windows only</td><td>Any (Docker)</td><td class="td-green">Cross-platform</td></tr>
  </table>
</div>


<div class="page">
  <div class="h1">Why We Made These Choices</div>

  <div class="h2">Why Go instead of Node.js?</div>
  <p class="p">The original backend was Express + TypeScript &mdash; 20 source files, npm dependencies, separate build step. Go compiles to a single binary with zero runtime dependencies. The entire backend + embedded frontend is one 15MB file. No node_modules, no npm install, no version conflicts. The Go stdlib provides HTTP server, JSON handling, and concurrent goroutines without external packages. We use exactly 3 Go dependencies: SQLite, WebPush, and the Anthropic SDK.</p>

  <div class="h2">Why build a custom CLI instead of using npm Claude CLI?</div>
  <p class="p">The npm Claude CLI was the single largest dependency &mdash; requiring Node.js in every agent container (adding ~260MB). It also had problems we couldn&rsquo;t fix: the <code>-p</code> flag exits after one prompt (no session persistence), the <code>--continue</code> flag was fragile, the first-run theme picker blocked non-interactive containers, and <code>--dangerously-skip-permissions</code> refused to run as root. Building our own CLI eliminated all of these. The Go binary starts in under 100ms, persists sessions natively, and integrates directly with the dashboard.</p>

  <div class="h2">Why Docker containers instead of host processes?</div>
  <p class="p">The original system spawned <code>wt.exe</code> terminal tabs &mdash; giving every agent full access to the host filesystem, network, and processes. A rogue agent prompt could delete files, exfiltrate data, or interfere with other agents. Docker containers provide: filesystem isolation (only project folder mounted), resource limits (memory + CPU caps), process isolation, and reproducible environments. The trade-off is no interactive terminal access, but the dashboard provides equivalent visibility.</p>

  <div class="h2">Why SSE instead of polling?</div>
  <p class="p">The old system polled every 3 seconds &mdash; wasting bandwidth and adding latency. SSE (Server-Sent Events) is a single HTTP connection held open by the server. When a message arrives, it&rsquo;s pushed to the agent instantly. The dashboard already had SSE for frontend updates; extending it to agents was trivial. We keep a 30-second fallback poll for resilience if the SSE connection drops.</p>

  <div class="h2">Why embed the frontend in the Go binary?</div>
  <p class="p">The old system needed a separate Nginx container just to serve static files and proxy API requests. Go&rsquo;s <code>embed</code> package lets us bake the compiled React app directly into the binary. One container, one port (9222), zero configuration. The frontend is built during the Docker multi-stage build &mdash; Node.js is only used at build time, never at runtime.</p>
</div>

<div class="page">
  <div class="h1">Pros of the New Approach</div>

  <div class="rec-list">
    <div class="rec-item">
      <div class="rec-num">1</div>
      <div class="rec-body"><strong>Security.</strong> Agents are sandboxed. Each container sees only its project folder. No host filesystem access, no network access beyond the dashboard API. Resource limits prevent runaway agents from consuming all system resources.</div>
    </div>
    <div class="rec-item">
      <div class="rec-num">2</div>
      <div class="rec-body"><strong>Portability.</strong> One prerequisite: Docker. Works on Windows, macOS, Linux. No Node.js, no Go, no Claude CLI to install on the host. The setup wizard handles everything else.</div>
    </div>
    <div class="rec-item">
      <div class="rec-num">3</div>
      <div class="rec-body"><strong>Control.</strong> 26 purpose-built tools means we define exactly what agents can do. Agent orchestration (spawn, relay, list) is native, not bash+curl hacks. Session persistence is built-in, not bolted on.</div>
    </div>
    <div class="rec-item">
      <div class="rec-num">4</div>
      <div class="rec-body"><strong>Efficiency.</strong> 49MB agent images (not 311MB). &lt;100ms message delivery (not 3s polling). &lt;100ms startup (not 3s Node.js init). 5&ndash;8 concurrent agents use under 500MB total.</div>
    </div>
    <div class="rec-item">
      <div class="rec-num">5</div>
      <div class="rec-body"><strong>Observability.</strong> All agent activity visible in the dashboard. File changes tracked automatically. Project timelines merge PM decisions with sub-agent updates. No more hunting through terminal tabs.</div>
    </div>
  </div>

  <div class="h1">Cons and Trade-offs</div>

  <div class="gap-list">
    <div class="gap-item">
      <div class="gap-num">1</div>
      <div class="rec-body"><strong>No Python/Node in default image.</strong> The 49MB image has bash, curl, git, jq &mdash; but not Python or Node.js. Agents must install runtimes via bash if needed, or a custom image must be built. This keeps the base image small but limits what agents can do out of the box.</div>
    </div>
    <div class="gap-item">
      <div class="gap-num">2</div>
      <div class="rec-body"><strong>No interactive terminal.</strong> Users cannot SSH into agent containers. All communication goes through dashboard messages and tool outputs. Good for security, harder for debugging.</div>
    </div>
    <div class="gap-item">
      <div class="gap-num">3</div>
      <div class="rec-body"><strong>OAuth bridge adds complexity.</strong> Enterprise OAuth tokens require a session+bridge handshake before API calls. Adds ~2 seconds to agent startup and requires careful token management.</div>
    </div>
    <div class="gap-item">
      <div class="gap-num">4</div>
      <div class="rec-body"><strong>Dashboard design is functional, not polished.</strong> The frontend was migrated from the original codebase. While the colour system was updated and legacy cruft removed, it needs a dedicated visual redesign to feel like a premium product.</div>
    </div>
    <div class="gap-item">
      <div class="gap-num">5</div>
      <div class="rec-body"><strong>Custom CLI means custom maintenance.</strong> Using the npm Claude CLI meant Anthropic maintained the tools. Our Go CLI means we maintain 26 tools. New Claude features (extended thinking, vision, etc.) must be added manually.</div>
    </div>
  </div>
</div>

"""

build_document(
    title='Claude Projects',
    subtitle='Technical Comparison: v3.0.0 vs New Architecture',
    brand='plain',
    layout='portrait',
    classification='non-confidential',
    cover_eyebrow='Technical Report &middot; April 2026',
    cover_pills=['Architecture Comparison', 'Go Rewrite', 'Container Agents', '26 Tools'],
    cover_bottom_bar=['Claude Projects', 'ClaudeAgentManager v3.0.0 vs Claude Projects'],
    content_html=CONTENT,
    output_name='Claude_Projects_Comparison_Report',
    output_dir=os.path.join(os.path.dirname(os.path.abspath(__file__)), 'output'),
)
