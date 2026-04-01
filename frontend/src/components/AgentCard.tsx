import { memo } from 'react';
import { useNavigate } from 'react-router-dom';
import { Activity, Clock, MessageSquare } from 'lucide-react';
import type { Agent } from '../types';
import { timeAgo } from '../utils/time';

const statusStyles: Record<string, { dot: string; label: string; pill: string }> = {
  active:              { dot: 'bg-emerald-400', label: 'Active', pill: 'text-emerald-400 bg-emerald-400/10 border-emerald-400/20' },
  working:             { dot: 'bg-blue-400 animate-pulse', label: 'Working', pill: 'text-blue-400 bg-blue-400/10 border-blue-400/20' },
  idle:                { dot: 'bg-amber-400', label: 'Idle', pill: 'text-amber-400 bg-amber-400/10 border-amber-400/20' },
  'waiting-for-input': { dot: 'bg-orange-400', label: 'Waiting', pill: 'text-orange-400 bg-orange-400/10 border-orange-400/20' },
  completed:           { dot: 'bg-white/30', label: 'Completed', pill: 'text-white/40 bg-white/5 border-white/10' },
  archived:            { dot: 'bg-white/20', label: 'Archived', pill: 'text-white/30 bg-white/[0.03] border-white/[0.06]' },
};

function AgentCard({ agent }: { agent: Agent }) {
  const navigate = useNavigate();
  const style = statusStyles[agent.status] || statusStyles.active;

  return (
    <button
      onClick={() => navigate(`/agent/${agent.id}`)}
      className="card-stagger group w-full text-left rounded-xl bg-surface-raised border border-white/[0.07] hover:border-white/[0.12] hover:bg-surface-overlay p-5 transition-all duration-150 focus-visible:outline-none relative"
    >
      {/* Unread badge */}
      {agent.unread_update_count > 0 && (
        <span className="absolute -top-1.5 -right-1.5 min-w-[20px] h-[20px] flex items-center justify-center rounded-full bg-accent-500 text-white text-[10px] font-bold z-10">
          {agent.unread_update_count}
        </span>
      )}

      {/* Header */}
      <div className="flex items-start justify-between gap-3 mb-3">
        <div className="min-w-0 flex-1">
          <h3 className="text-[14px] font-semibold text-white/90 truncate leading-tight">
            {agent.title || 'Untitled Agent'}
          </h3>
          {agent.role && (
            <span className="text-[11px] text-accent-400 font-medium">{agent.role}</span>
          )}
        </div>
        <span className={`chip shrink-0 ${style.pill}`}>
          <span className={`w-1.5 h-1.5 rounded-full ${style.dot}`} />
          {style.label}
        </span>
      </div>

      {/* Summary */}
      <p className={`text-[12px] leading-relaxed mb-4 line-clamp-2 ${
        agent.latest_summary ? 'text-white/50' : 'text-white/20 italic'
      }`}>
        {agent.latest_summary || 'No updates yet'}
      </p>

      {/* Stats row */}
      <div className="flex items-center gap-3 text-[11px] text-white/30">
        <span className="flex items-center gap-1">
          <Activity size={10} />
          {agent.update_count}
        </span>
        {agent.pending_message_count > 0 && (
          <span className="flex items-center gap-1 text-accent-400 font-medium">
            <MessageSquare size={10} />
            {agent.pending_message_count}
          </span>
        )}
        <span className="flex items-center gap-1 ml-auto">
          <Clock size={10} />
          {timeAgo(agent.last_update_at)}
        </span>
      </div>
    </button>
  );
}

export default memo(AgentCard);
