'use client';

import { useEffect, useState, useCallback, use } from 'react';
import Link from 'next/link';
import {
  fetchProject, fetchProjectAgents, fetchProjectUpdates, fetchProjectFiles,
  fetchAgentUpdates, fetchAgentMessages,
  startProject, pauseProject, completeProject, sendMessage,
} from '@/lib/api';
import { useSSE } from '@/providers/sse-provider';
import type { Project, Agent, ProjectUpdate, ProjectFile, TimelineEntry } from '@/types';
import { UnifiedTimeline, buildTimeline } from '@/components/unified-timeline';
import { MessageInput } from '@/components/message-input';
import { SpawnAgentDialog } from '@/components/spawn-agent-dialog';
import { StatusBadge } from '@/components/status-badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import {
  Play, Pause, CheckCircle, Bot, Plus, ChevronDown, ChevronRight,
  ArrowLeft, FileText,
} from 'lucide-react';
import { timeAgo } from '@/lib/time';

export default function ProjectDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const [project, setProject] = useState<Project | null>(null);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [updates, setUpdates] = useState<ProjectUpdate[]>([]);
  const [loading, setLoading] = useState(true);
  const [showSpawn, setShowSpawn] = useState(false);
  const [showArchived, setShowArchived] = useState(false);
  const [startPrompt, setStartPrompt] = useState('');
  const [files, setFiles] = useState<ProjectFile[]>([]);
  const [showStartForm, setShowStartForm] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { subscribe } = useSSE();

  const load = useCallback(async () => {
    try {
      const [p, a, u, f] = await Promise.all([
        fetchProject(id),
        fetchProjectAgents(id),
        fetchProjectUpdates(id, 500),
        fetchProjectFiles(id).catch(() => [] as ProjectFile[]),
      ]) as [any, any[], any[], ProjectFile[]];
      setProject(p);
      setAgents(a || []);
      setFiles(f || []);

      // Merge project updates with PM agent updates (thinking, tool calls, messages)
      let allUpdates = u || [];
      if (p?.pm_agent_id) {
        try {
          const pmUpdates = await fetchAgentUpdates(p.pm_agent_id, 500);
          const pmMessages = await fetchAgentMessages(p.pm_agent_id);
          // Add PM updates with source marker
          const pmTimeline = (pmUpdates || []).map((upd: any) => ({
            ...upd,
            _source: 'pm_agent',
            timestamp: upd.timestamp || upd.created_at,
          }));
          // Add PM relay messages as timeline entries
          const pmMsgs = (pmMessages || []).filter((m: any) => m.source_agent_id).map((m: any) => ({
            id: m.id + 100000,
            type: 'message' as const,
            content: m.content,
            timestamp: m.created_at,
            _source: 'pm_message',
            _source_agent_id: m.source_agent_id,
          }));
          allUpdates = [...allUpdates, ...pmTimeline, ...pmMsgs];
        } catch {
          // PM updates fetch failed — use project updates only
        }
      }

      // Sort by timestamp
      allUpdates.sort((a: any, b: any) => {
        const ta = a.timestamp || a.created_at || '';
        const tb = b.timestamp || b.created_at || '';
        return ta.localeCompare(tb);
      });
      setUpdates(allUpdates);
    } catch (err: any) {
      console.error('Failed to load project:', err);
      setError(err?.message || 'Failed to load project');
      setTimeout(() => setError(null), 5000);
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => { load(); }, [load]);

  // SSE subscriptions
  useEffect(() => {
    const unsub1 = subscribe('project-updated', (p: Project) => {
      if (p.id === id) setProject(p);
    });
    const unsub2 = subscribe('agent-updated', (a: Agent) => {
      if (a.project_id === id) {
        setAgents(prev => {
          const idx = prev.findIndex(x => x.id === a.id);
          if (idx >= 0) { const next = [...prev]; next[idx] = a; return next; }
          return [...prev, a];
        });
      }
    });
    return () => { unsub1(); unsub2(); };
  }, [subscribe, id]);

  const handleStart = useCallback(async () => {
    if (!project) return;
    try {
      await startProject(id, startPrompt);
      setShowStartForm(false);
      setStartPrompt('');
      load();
    } catch (err: any) {
      console.error('Failed to start project:', err);
      setError(err?.message || 'Failed to start project');
      setTimeout(() => setError(null), 5000);
    }
  }, [id, startPrompt, project, load]);

  const handlePause = useCallback(async () => {
    try { await pauseProject(id); load(); } catch (err: any) { console.error('Failed to pause project:', err); setError(err?.message || 'Failed to pause'); setTimeout(() => setError(null), 5000); }
  }, [id, load]);

  const handleComplete = useCallback(async () => {
    try { await completeProject(id); load(); } catch (err: any) { console.error('Failed to complete project:', err); setError(err?.message || 'Failed to complete'); setTimeout(() => setError(null), 5000); }
  }, [id, load]);

  const handleSendMessage = useCallback(async (content: string) => {
    if (!project?.pm_agent_id) return;
    await sendMessage(project.pm_agent_id, content);
  }, [project]);

  if (loading) return <div className="p-8 text-muted-foreground">Loading...</div>;
  if (!project) return <div className="p-8 text-muted-foreground">Project not found.</div>;

  const activeAgents = agents.filter(a => !['archived', 'completed'].includes(a.status));
  const archivedAgents = agents.filter(a => ['archived', 'completed'].includes(a.status));

  // Split updates: agent-level updates vs project-level updates
  const agentUpdates = updates.filter((u: any) => u._source === 'pm_agent' || u._source === 'pm_message');
  const projectOnlyUpdates = updates.filter((u: any) => !u._source);
  const timeline: TimelineEntry[] = buildTimeline(agentUpdates as any[], undefined, projectOnlyUpdates as any[]);

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="border-b border-border px-6 py-4 bg-card shrink-0">
        <div className="flex items-center gap-3 mb-2">
          <Link href="/projects" className="text-muted-foreground hover:text-foreground">
            <ArrowLeft size={18} />
          </Link>
          <h1 className="text-xl font-bold text-foreground">{project.name}</h1>
          <StatusBadge status={project.status} />
        </div>
        {project.description && (
          <p className="text-sm text-muted-foreground mb-3 ml-8">{project.description}</p>
        )}
        <div className="flex items-center gap-2 ml-8">
          {project.status === 'pending' && (
            <>
              {showStartForm ? (
                <div className="flex items-center gap-2 flex-1">
                  <input
                    type="text"
                    value={startPrompt}
                    onChange={e => setStartPrompt(e.target.value)}
                    placeholder="Initial prompt for PM (optional)..."
                    className="flex-1 rounded-lg border border-input bg-background px-3 py-1.5 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                    onKeyDown={e => e.key === 'Enter' && handleStart()}
                    autoFocus
                  />
                  <Button size="sm" onClick={handleStart}><Play size={14} /> Start</Button>
                  <Button size="sm" variant="outline" onClick={() => setShowStartForm(false)}>Cancel</Button>
                </div>
              ) : (
                <Button size="sm" onClick={() => setShowStartForm(true)}><Play size={14} /> Start Project</Button>
              )}
            </>
          )}
          {project.status === 'active' && (
            <>
              <Button size="sm" variant="outline" onClick={handlePause}><Pause size={14} /> Pause</Button>
              <Button size="sm" variant="outline" onClick={handleComplete}><CheckCircle size={14} /> Complete</Button>
              <Button size="sm" onClick={() => setShowSpawn(true)}><Plus size={14} /> Spawn Agent</Button>
            </>
          )}
          {project.status === 'paused' && (
            <Button size="sm" onClick={() => { startProject(id); load(); }}><Play size={14} /> Resume</Button>
          )}
        </div>
      </div>

      {/* Error banner */}
      {error && <div className="px-6 py-2 text-destructive text-sm bg-destructive/5 border-b border-destructive/10">{error}</div>}

      {/* Main content */}
      <div className="flex-1 overflow-auto">
        <div className="max-w-5xl mx-auto p-6 space-y-6">
          {/* Active agents grid */}
          {activeAgents.length > 0 && (
            <div>
              <h2 className="text-sm font-semibold text-foreground mb-3 flex items-center gap-2">
                <Bot size={16} /> Active Agents ({activeAgents.length})
              </h2>
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
                {activeAgents.map(agent => (
                  <AgentCard key={agent.id} agent={agent} />
                ))}
              </div>
            </div>
          )}

          {/* Archived agents */}
          {archivedAgents.length > 0 && (
            <div>
              <button
                onClick={() => setShowArchived(!showArchived)}
                className="text-sm font-semibold text-muted-foreground hover:text-foreground flex items-center gap-1.5"
              >
                {showArchived ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
                Archived Agents ({archivedAgents.length})
              </button>
              {showArchived && (
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 mt-3">
                  {archivedAgents.map(agent => (
                    <AgentCard key={agent.id} agent={agent} />
                  ))}
                </div>
              )}
            </div>
          )}

          {/* Timeline — newest first */}
          <div>
            <h2 className="text-sm font-semibold text-foreground mb-3">Timeline</h2>
            <Card>
              <CardContent className="p-0">
                <UnifiedTimeline entries={[...timeline].reverse()} className="py-3" autoScroll={false} />
              </CardContent>
            </Card>
          </div>

          {/* Files */}
          {files.length > 0 && (
            <div>
              <h2 className="text-sm font-semibold text-foreground mb-3 flex items-center gap-2">
                <FileText size={16} /> Files ({files.length})
              </h2>
              <Card>
                <CardContent className="p-0">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-border text-left text-muted-foreground">
                        <th className="px-4 py-2 font-medium">Filename</th>
                        <th className="px-4 py-2 font-medium">Size</th>
                        <th className="px-4 py-2 font-medium">Agent</th>
                        <th className="px-4 py-2 font-medium text-right">Modified</th>
                      </tr>
                    </thead>
                    <tbody>
                      {files.map((file: any, i: number) => (
                        <tr key={i} className="border-b border-border last:border-0 hover:bg-accent/50">
                          <td className="px-4 py-2 text-foreground font-mono text-xs">{file.filename || file.name || '—'}</td>
                          <td className="px-4 py-2 text-muted-foreground">{formatFileSize(file.size || 0)}</td>
                          <td className="px-4 py-2 text-muted-foreground">{file.agent_role || file.agent_title || '—'}</td>
                          <td className="px-4 py-2 text-muted-foreground text-right">{file.created_at ? timeAgo(file.created_at) : '—'}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </CardContent>
              </Card>
            </div>
          )}
        </div>
      </div>

      {/* Message input for PM */}
      {project.pm_agent_id && project.status === 'active' && (
        <MessageInput
          onSend={handleSendMessage}
          placeholder="Send a message to the PM..."
        />
      )}

      {/* Spawn dialog */}
      <SpawnAgentDialog
        projectId={id}
        open={showSpawn}
        onClose={() => setShowSpawn(false)}
        onSpawned={() => load()}
      />
    </div>
  );
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB'];
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  const value = bytes / Math.pow(1024, i);
  return `${value < 10 && i > 0 ? value.toFixed(1) : Math.round(value)} ${units[i]}`;
}

function AgentCard({ agent }: { agent: Agent }) {
  const statusColors: Record<string, string> = {
    active: 'bg-green-500',
    working: 'bg-blue-500 animate-pulse',
    idle: 'bg-yellow-500',
    'waiting-for-input': 'bg-orange-500',
    completed: 'bg-gray-400',
    archived: 'bg-gray-300',
  };

  return (
    <Link href={`/agent/${agent.id}`}>
      <Card className="hover:bg-accent/50 transition-colors cursor-pointer">
        <CardContent className="p-4">
          <div className="flex items-center gap-2 mb-1.5">
            <span className={`w-2 h-2 rounded-full shrink-0 ${statusColors[agent.status] || 'bg-gray-400'}`} />
            <span className="text-xs text-muted-foreground capitalize">{agent.status.replace(/-/g, ' ')}</span>
            <span className="text-xs text-muted-foreground ml-auto">{timeAgo(agent.last_update_at || agent.created_at)}</span>
          </div>
          <p className="font-medium text-foreground text-sm truncate">{agent.title || agent.role || 'Agent'}</p>
          {agent.role && agent.role !== agent.title && (
            <p className="text-xs text-primary truncate">{agent.role}</p>
          )}
          {agent.latest_summary && (
            <p className="text-xs text-muted-foreground mt-1 line-clamp-2">{agent.latest_summary}</p>
          )}
        </CardContent>
      </Card>
    </Link>
  );
}
