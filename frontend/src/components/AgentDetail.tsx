import { useParams, useNavigate } from 'react-router-dom';
import { useState, useEffect } from 'react';
import { ArrowLeft, Calendar, Activity, Archive, ArchiveRestore, FileDown, Play, XCircle } from 'lucide-react';
import { useAgent } from '../hooks/useAgent';
import { updateAgent, markAgentRead, createLaunchRequest, closeAgent } from '../api';
import { formatDate } from '../utils/time';
import UpdateTimeline from './UpdateTimeline';
import MessagePanel from './MessagePanel';
import ProjectTodoPanel from './ProjectTodoPanel';
import FilesPanel from './FilesPanel';
import type { ProjectStatus, TodoStatus } from '../types';

const statusConfig = {
  active: { color: 'bg-emerald-400', label: 'Active' },
  working: { color: 'bg-blue-400', label: 'Working' },
  idle: { color: 'bg-amber-400', label: 'Idle' },
  'waiting-for-input': { color: 'bg-orange-400', label: 'Waiting for Input' },
  completed: { color: 'bg-white/30', label: 'Completed' },
  archived: { color: 'bg-white/20', label: 'Archived' },
} as const;

function AgentDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { agent, updates, messages, loading, error, refetch } = useAgent(id!);

  const isArchived = agent?.status === 'archived';
  const [exporting, setExporting] = useState(false);
  const [resuming, setResuming] = useState(false);
  const [closing, setClosing] = useState(false);

  // Mark agent as read when viewing detail page
  useEffect(() => {
    if (id && agent) {
      markAgentRead(id).catch(() => {});
    }
  }, [id, agent?.update_count]);

  const handleToggleArchive = async () => {
    if (!id || !agent) return;
    try {
      await updateAgent(id, { status: isArchived ? 'active' : 'archived' });
      refetch();
    } catch {
      // Error will be shown through the hook
    }
  };

  if (loading) {
    return (
      <div className="max-w-7xl mx-auto px-6 py-8">
        <div className="animate-pulse">
          <div className="h-8 w-32 bg-surface-raised rounded mb-6" />
          <div className="h-10 w-64 bg-surface-raised rounded mb-4" />
          <div className="h-4 w-48 bg-surface-raised rounded mb-8" />
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            <div className="lg:col-span-2 h-96 bg-surface-raised rounded-xl border border-white/[0.07]" />
            <div className="h-96 bg-surface-raised rounded-xl border border-white/[0.07]" />
          </div>
        </div>
      </div>
    );
  }

  if (error || !agent) {
    return (
      <div className="max-w-7xl mx-auto px-6 py-8">
        <button
          onClick={() => navigate('/')}
          className="inline-flex items-center gap-2 text-white/40 hover:text-white/70 mb-6 transition-colors text-sm"
        >
          <ArrowLeft size={16} />
          Back to dashboard
        </button>
        <div className="rounded-xl bg-red-950/20 border border-red-500/20 p-6 text-center">
          <p className="text-red-400">{error || 'Agent not found'}</p>
        </div>
      </div>
    );
  }

  const status = statusConfig[agent.status];

  let parsedProjects: ProjectStatus[] = [];
  let parsedTodos: TodoStatus[] = [];
  try {
    const raw = agent.metadata;
    const meta = typeof raw === 'string' ? JSON.parse(raw) : (raw || {});
    if (Array.isArray(meta.projects)) parsedProjects = meta.projects;
    if (Array.isArray(meta.todos)) parsedTodos = meta.todos;
  } catch { /* ignore parse errors */ }
  const projects = parsedProjects;
  const todos = parsedTodos;

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 py-4 sm:py-8 page-enter">
      {/* Back button */}
      <button
        onClick={() => navigate('/')}
        className="inline-flex items-center gap-2 text-white/40 hover:text-white/70 mb-4 sm:mb-6 transition-colors text-sm"
      >
        <ArrowLeft size={16} />
        Back to dashboard
      </button>

      {/* Agent header */}
      <div className="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4 mb-8">
        <div className="min-w-0">
          <div className="flex items-center gap-3 mb-2 flex-wrap">
            <h1 className="text-xl sm:text-2xl font-bold text-white/90 truncate">{agent.title}</h1>
            <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full border border-white/[0.07] bg-surface-raised shrink-0">
              <span className={`w-2 h-2 rounded-full ${status.color} ${agent.status === 'active' ? 'animate-pulse' : ''}`} />
              <span className="text-xs font-medium text-white/60">{status.label}</span>
            </span>
          </div>
          <div className="flex items-center gap-4 text-sm text-white/35 flex-wrap">
            <span className="flex items-center gap-1.5">
              <Calendar size={14} />
              Created {formatDate(agent.created_at)}
            </span>
            <span className="flex items-center gap-1.5">
              <Activity size={14} />
              {agent.update_count} update{agent.update_count !== 1 ? 's' : ''}
            </span>
          </div>
        </div>

        <div className="flex items-center gap-1 shrink-0">
          <button
            onClick={async () => {
              if (!id || !agent || resuming) return;
              setResuming(true);
              try {
                const cwdPath = (agent.cwd || '').replace(/\\/g, '/');
                await createLaunchRequest('resume', cwdPath || agent.workspace || '', id);
              } catch (err) {
                console.error('Resume failed:', err);
              } finally {
                setResuming(false);
              }
            }}
            disabled={resuming}
            className="inline-flex items-center gap-2 px-3 py-2 text-sm text-white/40 hover:text-emerald-400 hover:bg-emerald-500/10 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            title="Resume this agent session"
          >
            <Play size={16} />
            <span className="text-xs">{resuming ? 'Resuming...' : 'Resume'}</span>
          </button>
          <button
            onClick={async () => {
              if (!id || exporting) return;
              setExporting(true);
              try {
                const res = await fetch(`/api/agents/${id}/export/pdf`);
                if (!res.ok) throw new Error('Export failed');
                const blob = await res.blob();
                const url = URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = url;
                a.download = `Agent_Report_${id.slice(0, 8)}.pdf`;
                a.click();
                URL.revokeObjectURL(url);
              } catch (err) {
                console.error('PDF export failed:', err);
              } finally {
                setExporting(false);
              }
            }}
            disabled={exporting}
            className="inline-flex items-center gap-2 px-3 py-2 text-sm text-white/40 hover:text-accent-400 hover:bg-accent-500/10 rounded-lg transition-colors disabled:opacity-50"
            title="Export as PDF"
          >
            <FileDown size={16} />
            <span className="text-xs">{exporting ? 'Exporting...' : 'PDF'}</span>
          </button>
          <button
            onClick={handleToggleArchive}
            className={`inline-flex items-center gap-2 px-3 py-2 text-sm rounded-lg transition-colors ${
              isArchived
                ? 'text-white/40 hover:text-emerald-400 hover:bg-emerald-500/10'
                : 'text-white/40 hover:text-amber-400 hover:bg-amber-500/10'
            }`}
            title={isArchived ? 'Unarchive agent' : 'Archive agent'}
          >
            {isArchived ? <ArchiveRestore size={16} /> : <Archive size={16} />}
            <span className="text-xs">{isArchived ? 'Unarchive' : 'Archive'}</span>
          </button>
          <button
            onClick={async () => {
              if (!id || !agent || closing) return;
              if (!confirm('Close this agent? This will archive it and terminate its Claude process.')) return;
              setClosing(true);
              try {
                const result = await closeAgent(id);
                if (!result.terminated) {
                  alert('Agent archived, but no PID was stored — the Claude process may still be running.');
                }
                refetch();
              } catch (err) {
                console.error('Close failed:', err);
              } finally {
                setClosing(false);
              }
            }}
            disabled={closing || isArchived}
            className="inline-flex items-center gap-2 px-3 py-2 text-sm text-white/40 hover:text-red-400 hover:bg-red-500/10 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            title="Close agent — archive and terminate process"
          >
            <XCircle size={16} />
            <span className="text-xs">{closing ? 'Closing...' : 'Close'}</span>
          </button>
        </div>
      </div>

      {/* Layout: on mobile stack messages -> projects/todos -> timeline; on desktop 3-col */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4 sm:gap-6">
        {/* Messages — first on mobile, sidebar on desktop */}
        <div className="order-1 lg:order-2 space-y-4">
          <MessagePanel agentId={agent.id} messages={messages} onSent={refetch} />
        </div>
        {/* Projects & Todos — second on mobile */}
        <div className="lg:col-span-2 order-2 lg:order-3 space-y-4">
          <ProjectTodoPanel projects={projects} todos={todos} />
          <FilesPanel agentId={agent.id} />
        </div>
        {/* Timeline — last on mobile, main area on desktop */}
        <div className="lg:col-span-2 order-3 lg:order-1">
          <UpdateTimeline updates={updates} />
        </div>
      </div>
    </div>
  );
}

export default AgentDetail;
