'use client';
import { useEffect, useState } from 'react';
import Link from 'next/link';
import { fetchProjects, fetchAgents, fetchRolesStats, closeAgent } from '@/lib/api';
import { useSSE } from '@/providers/sse-provider';
import { StatusBadge } from '@/components/status-badge';
import { FolderKanban, Bot, Brain, Activity, ArrowRight, Plus } from 'lucide-react';
import { timeAgo } from '@/lib/time';
import type { Project, Agent } from '@/types';

export default function DashboardPage() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [roleCount, setRoleCount] = useState(0);
  const [loading, setLoading] = useState(true);
  const { state: sseState, subscribe } = useSSE();

  useEffect(() => {
    Promise.all([
      fetchProjects().catch(() => []),
      fetchAgents().catch(() => []),
      fetchRolesStats().catch(() => ({ total_roles: 0 })),
    ]).then(([p, a, r]) => {
      setProjects(p);
      setAgents(a);
      setRoleCount((r as any).total_roles || 0);
      setLoading(false);
    });
  }, []);

  // SSE subscriptions to keep dashboard live
  useEffect(() => {
    const unsub1 = subscribe('project-updated', (p: Project) => {
      setProjects(prev => {
        const idx = prev.findIndex(x => x.id === p.id);
        if (idx >= 0) { const next = [...prev]; next[idx] = p; return next; }
        return [p, ...prev];
      });
    });
    const unsub2 = subscribe('project-created', (p: Project) => {
      setProjects(prev => [p, ...prev]);
    });
    const unsub3 = subscribe('agent-updated', (a: Agent) => {
      setAgents(prev => {
        const idx = prev.findIndex(x => x.id === a.id);
        if (idx >= 0) { const next = [...prev]; next[idx] = a; return next; }
        return [...prev, a];
      });
    });
    return () => { unsub1(); unsub2(); unsub3(); };
  }, [subscribe]);

  if (loading) return <div className="p-8 text-muted-foreground">Loading...</div>;

  const activeProjects = projects.filter(p => p.status === 'active');
  const activeAgents = agents.filter(a => ['active', 'working'].includes(a.status));
  const totalFiles = agents.reduce((sum, a) => sum + (a.update_count || 0), 0);

  return (
    <div className="p-8 max-w-6xl">
      <h1 className="text-2xl font-bold text-foreground mb-6">Dashboard</h1>

      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
        <div className="p-4 rounded-lg border border-border bg-card">
          <div className="flex items-center gap-2 text-muted-foreground mb-1">
            <FolderKanban size={14} />
            <span className="text-xs">Active Projects</span>
          </div>
          <p className="text-2xl font-bold text-foreground">{activeProjects.length}</p>
        </div>
        <div className="p-4 rounded-lg border border-border bg-card">
          <div className="flex items-center gap-2 text-muted-foreground mb-1">
            <Bot size={14} />
            <span className="text-xs">Running Agents</span>
          </div>
          <p className="text-2xl font-bold text-foreground">{activeAgents.length}</p>
        </div>
        <div className="p-4 rounded-lg border border-border bg-card">
          <div className="flex items-center gap-2 text-muted-foreground mb-1">
            <Brain size={14} />
            <span className="text-xs">Role Library</span>
          </div>
          <p className="text-2xl font-bold text-foreground">{roleCount}</p>
        </div>
        <div className="p-4 rounded-lg border border-border bg-card">
          <div className="flex items-center gap-2 text-muted-foreground mb-1">
            <Activity size={14} />
            <span className="text-xs">Connection</span>
          </div>
          <p className={`text-2xl font-bold ${sseState === 'connected' ? 'text-green-600' : 'text-red-500'}`}>
            {sseState === 'connected' ? 'Live' : 'Offline'}
          </p>
        </div>
      </div>

      {/* Active Projects */}
      <div className="mb-8">
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-lg font-semibold text-foreground">Projects</h2>
          <Link
            href="/projects"
            className="text-xs text-primary hover:underline flex items-center gap-1"
          >
            View all <ArrowRight size={12} />
          </Link>
        </div>
        {projects.length === 0 ? (
          <div className="text-center py-12 border border-dashed border-border rounded-lg">
            <FolderKanban size={32} className="mx-auto text-muted-foreground/40 mb-3" />
            <p className="text-sm text-muted-foreground mb-3">No projects yet</p>
            <Link
              href="/projects"
              className="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-md bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
            >
              <Plus size={14} /> Create Project
            </Link>
          </div>
        ) : (
          <div className="space-y-2">
            {projects.slice(0, 5).map(project => {
              const projectAgents = agents.filter(a => a.project_id === project.id);
              const runningCount = projectAgents.filter(a => ['active', 'working'].includes(a.status)).length;
              return (
                <Link
                  key={project.id}
                  href={`/projects/${project.id}`}
                  className="flex items-center gap-4 p-4 rounded-lg border border-border bg-card hover:bg-accent transition-colors"
                >
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <p className="font-medium text-foreground truncate">{project.name}</p>
                      <StatusBadge status={project.status} />
                    </div>
                    <p className="text-xs text-muted-foreground truncate">{project.description}</p>
                  </div>
                  <div className="flex items-center gap-4 shrink-0 text-xs text-muted-foreground">
                    {runningCount > 0 && (
                      <span className="flex items-center gap-1">
                        <Bot size={12} className="text-green-500" />
                        {runningCount} running
                      </span>
                    )}
                    {project.started_at && (
                      <span>{timeAgo(project.started_at)}</span>
                    )}
                  </div>
                  <ArrowRight size={16} className="text-muted-foreground/40 shrink-0" />
                </Link>
              );
            })}
          </div>
        )}
      </div>

      {/* Running Agents — only from active projects */}
      {(() => {
        const activeProjectIds = new Set(activeProjects.map(p => p.id));
        const projectAgents = activeAgents.filter(a => a.project_id && activeProjectIds.has(a.project_id));
        const orphanedAgents = activeAgents.filter(a => !a.project_id || !activeProjectIds.has(a.project_id));

        return (
          <>
            {projectAgents.length > 0 && (
              <div className="mb-8">
                <h2 className="text-lg font-semibold text-foreground mb-3">Running Agents</h2>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
                  {projectAgents.slice(0, 6).map(agent => (
                    <Link
                      key={agent.id}
                      href={`/agent/${agent.id}`}
                      className="p-3 rounded-lg border border-border bg-card hover:bg-accent transition-colors"
                    >
                      <div className="flex items-center gap-2 mb-1">
                        <span className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
                        <span className="text-xs text-muted-foreground capitalize">{agent.status}</span>
                        {agent.role && <span className="text-xs text-primary ml-auto">{agent.role}</span>}
                      </div>
                      <p className="text-sm font-medium text-foreground truncate">{agent.title || agent.role || 'Agent'}</p>
                    </Link>
                  ))}
                </div>
              </div>
            )}
            {orphanedAgents.length > 0 && (
              <div>
                <div className="flex items-center justify-between mb-3">
                  <h2 className="text-lg font-semibold text-muted-foreground">Orphaned Agents ({orphanedAgents.length})</h2>
                  <button
                    onClick={async () => {
                      for (const a of orphanedAgents) {
                        await closeAgent(a.id).catch(() => {});
                      }
                      window.location.reload();
                    }}
                    className="text-xs text-destructive hover:underline"
                  >
                    Close all orphaned
                  </button>
                </div>
                <p className="text-xs text-muted-foreground mb-3">These agents have no active project. Close them to free resources.</p>
              </div>
            )}
          </>
        );
      })()}
    </div>
  );
}
