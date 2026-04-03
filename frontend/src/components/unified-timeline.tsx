'use client';

import { useRef, useEffect } from 'react';
import type { TimelineEntry, AgentUpdate, AgentMessage, ProjectUpdate } from '@/types';
import { formatTime, parseDate } from '@/lib/time';
import { Wrench, Info, Milestone, BrainCircuit, AlertTriangle, Bot, CheckCircle, MessageSquare } from 'lucide-react';

interface UnifiedTimelineProps {
  entries: TimelineEntry[];
  className?: string;
  autoScroll?: boolean;
}

export function UnifiedTimeline({ entries, className = '', autoScroll = true }: UnifiedTimelineProps) {
  const bottomRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (autoScroll && bottomRef.current) {
      bottomRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [entries.length, autoScroll]);

  if (entries.length === 0) {
    return (
      <div className={`flex items-center justify-center py-12 text-muted-foreground text-sm ${className}`}>
        No activity yet.
      </div>
    );
  }

  return (
    <div ref={containerRef} className={`space-y-1 ${className}`}>
      {entries.map((entry, i) => (
        <TimelineItem key={`${entry.kind}-${getEntryId(entry)}-${i}`} entry={entry} />
      ))}
      <div ref={bottomRef} />
    </div>
  );
}

function getEntryId(entry: TimelineEntry): string | number {
  if (entry.kind === 'update') return entry.data.id;
  if (entry.kind === 'message') return entry.data.id;
  if (entry.kind === 'project_update') return entry.data.id;
  return 0;
}

function TimelineItem({ entry }: { entry: TimelineEntry }) {
  if (entry.kind === 'update') return <UpdateItem update={entry.data} />;
  if (entry.kind === 'message') return <MessageItem message={entry.data} />;
  if (entry.kind === 'project_update') return <ProjectUpdateItem update={entry.data} />;
  return null;
}

// Extract readable text from content that might be JSON-wrapped
function parseContent(content: string): string {
  if (!content) return '';
  try {
    const parsed = JSON.parse(content);
    if (typeof parsed === 'string') return parsed;
    if (typeof parsed === 'object') {
      // {"text":"..."} → extract text
      if (parsed.text) return String(parsed.text);
      // {"output":"..."} → task completion
      if (parsed.output !== undefined) {
        const parts: string[] = [];
        if (parsed.output) parts.push(String(parsed.output));
        if (parsed.files_changed && parsed.files_changed !== 'No files modified') parts.push(`Files: ${parsed.files_changed}`);
        if (parsed.estimated_cost) parts.push(`Cost: ${parsed.estimated_cost}`);
        return parts.join('\n');
      }
      // {"type":"...","content":"..."} → post_update
      if (parsed.type && parsed.content) return String(parsed.content);
      // {"status":"..."} → status
      if (parsed.status) return String(parsed.status);
      // {"path":"..."} → file operation
      if (parsed.path) return String(parsed.path);
      // {"key":"...","content":"..."} → scratchpad
      if (parsed.key && parsed.content) return `${parsed.key}: ${String(parsed.content).slice(0, 200)}`;
      if (parsed.key) return String(parsed.key);
      // {"role":"...","prompt":"..."} → spawn agent
      if (parsed.role && parsed.prompt) return `${parsed.role}\n${String(parsed.prompt).slice(0, 300)}`;
      if (parsed.role) return String(parsed.role);
      // {"command":"..."} → bash
      if (parsed.command) return `$ ${parsed.command}`;
      // {"query":"..."} → web_search
      if (parsed.query) return `search: ${parsed.query}`;
      // {"target_agent_id":"...","content":"..."} → relay_message
      if (parsed.target_agent_id && parsed.content) return String(parsed.content).slice(0, 300);
      // Fallback — pick the most interesting field
      const keys = Object.keys(parsed);
      if (keys.length <= 3) {
        return keys.map(k => `${k}: ${String(parsed[k]).slice(0, 100)}`).join(', ');
      }
      return keys.slice(0, 3).map(k => `${k}: ${String(parsed[k]).slice(0, 60)}`).join(', ') + '...';
    }
    return String(parsed);
  } catch {
    return content;
  }
}

// Format tool call content into a clean one-liner
function formatToolDetail(content: string): string {
  if (!content) return '';
  try {
    const parsed = JSON.parse(content);
    if (typeof parsed === 'string') return parsed;
    if (typeof parsed === 'object') {
      if (parsed.path) return parsed.path;
      if (parsed.command) return `$ ${parsed.command.slice(0, 80)}`;
      if (parsed.query) return parsed.query;
      if (parsed.key) return parsed.key;
      if (parsed.role) return parsed.role;
      if (parsed.pattern) return parsed.pattern;
      if (parsed.content && parsed.target_agent_id) return `→ ${parsed.content.slice(0, 80)}`;
      if (parsed.agent_id) return parsed.agent_id.slice(0, 12);
      if (parsed.url) return parsed.url;
      const keys = Object.keys(parsed).filter(k => parsed[k]);
      if (keys.length === 0) return '';
      return keys.slice(0, 2).map(k => `${String(parsed[k]).slice(0, 40)}`).join(' ');
    }
    return '';
  } catch {
    return content.slice(0, 80);
  }
}

// Escape HTML entities to prevent XSS via dangerouslySetInnerHTML
function escapeHtml(text: string): string {
  return text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

// Render text with basic markdown: **bold**, newlines, bullet points
function FormattedText({ text, className = '' }: { text: string; className?: string }) {
  if (!text) return null;
  // Split into paragraphs on double newline
  const paragraphs = text.split(/\n{2,}/);
  return (
    <div className={className}>
      {paragraphs.map((para, i) => {
        // Split single newlines into lines
        const lines = para.split('\n');
        return (
          <div key={i} className={i > 0 ? 'mt-2' : ''}>
            {lines.map((line, j) => {
              const trimmed = line.trim();
              if (!trimmed) return null;
              // Bullet points
              const isBullet = /^[-*•]\s/.test(trimmed) || /^\d+\.\s/.test(trimmed);
              // Escape HTML first, then apply markdown transforms
              const escaped = escapeHtml(trimmed);
              const html = escaped
                .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
                .replace(/\*(.+?)\*/g, '<em>$1</em>')
                .replace(/`(.+?)`/g, '<code class="bg-gray-100 px-1 rounded text-xs">$1</code>');
              return (
                <div
                  key={j}
                  className={isBullet ? 'pl-3' : ''}
                  dangerouslySetInnerHTML={{ __html: html }}
                />
              );
            })}
          </div>
        );
      })}
    </div>
  );
}

function UpdateItem({ update }: { update: AgentUpdate }) {
  if (update.type === 'thinking') {
    const text = parseContent(update.content);
    return (
      <div className="py-1.5 px-3">
        <div className="bg-violet-50 border border-violet-100 rounded-lg px-3 py-2 text-sm">
          <div className="flex items-start justify-between gap-2">
            <div className="flex items-start gap-2 flex-1">
              <BrainCircuit size={14} className="mt-0.5 shrink-0 text-violet-400" />
              <FormattedText text={text} className="flex-1 text-foreground/80 italic leading-relaxed" />
            </div>
            <span className="text-[10px] text-muted-foreground shrink-0 mt-0.5">{formatTime(update.timestamp)}</span>
          </div>
        </div>
      </div>
    );
  }

  if (update.type === 'tool') {
    const toolName = (update as any).summary || update.metadata || '';
    const detail = formatToolDetail(update.content);
    return (
      <div className="flex items-center gap-2 py-0.5 px-3 text-xs text-muted-foreground font-mono group hover:bg-secondary/30 rounded">
        <Wrench size={11} className="shrink-0 text-muted-foreground/50" />
        <span className="text-foreground/60 font-semibold shrink-0">{toolName}</span>
        <span className="flex-1 truncate text-muted-foreground/40">{detail}</span>
        <span className="text-[10px] shrink-0 opacity-0 group-hover:opacity-100 transition-opacity">
          {formatTime(update.timestamp)}
        </span>
      </div>
    );
  }

  if (update.type === 'status') {
    const statusText = parseContent(update.content);
    return (
      <div className="flex items-center gap-2 py-0.5 px-3 text-[11px] text-muted-foreground/50">
        <span className="w-1 h-1 rounded-full bg-muted-foreground/30 shrink-0" />
        <span className="flex-1">{statusText}</span>
        <span className="text-[10px] shrink-0">{formatTime(update.timestamp)}</span>
      </div>
    );
  }

  if (update.type === 'error') {
    return (
      <div className="flex items-start gap-2 py-1.5 px-3 bg-destructive/5 rounded text-sm border border-destructive/10">
        <AlertTriangle size={14} className="mt-0.5 shrink-0 text-destructive" />
        <span className="flex-1 text-destructive/90">{update.content}</span>
        <span className="text-[10px] shrink-0 text-muted-foreground">{formatTime(update.timestamp)}</span>
      </div>
    );
  }

  if (update.type === 'progress') {
    return (
      <div className="flex items-start gap-2 py-1 px-3 text-xs text-muted-foreground">
        <div className="w-3 h-3 rounded-full border-2 border-primary/40 border-t-primary animate-spin mt-0.5 shrink-0" />
        <span className="flex-1">{update.content}</span>
        <span className="text-[10px] shrink-0">{formatTime(update.timestamp)}</span>
      </div>
    );
  }

  // type === 'text' or 'diagram' or 'info' - render as card
  const displayContent = parseContent(update.content);

  return (
    <div className="py-1.5 px-3">
      <div className="bg-secondary/40 rounded-lg px-3 py-2 text-sm">
        <div className="flex items-start justify-between gap-2">
          <FormattedText text={displayContent} className="flex-1 break-words" />
          <span className="text-[10px] text-muted-foreground shrink-0 mt-0.5">{formatTime(update.timestamp)}</span>
        </div>
      </div>
    </div>
  );
}

function MessageItem({ message }: { message: AgentMessage }) {
  const isUser = message.role === 'user';

  return (
    <div className={`py-1.5 px-3 flex ${isUser ? 'justify-end' : 'justify-start'}`}>
      <div
        className={`max-w-[80%] rounded-lg px-3 py-2 text-sm ${
          isUser
            ? 'bg-primary text-primary-foreground'
            : 'bg-secondary/60 text-foreground'
        }`}
      >
        {!isUser && message.source_agent_id && (
          <div className="text-[10px] text-primary/70 mb-1 flex items-center gap-1 font-medium">
            <Bot size={10} />
            {(message as any).source_agent_title || `Agent ${message.source_agent_id.slice(0, 8)}`}
          </div>
        )}
        {isUser && !message.source_agent_id && (
          <div className="text-[10px] text-primary-foreground/70 mb-1 font-medium">You</div>
        )}
        {isUser && message.source_agent_id && (
          <div className="text-[10px] text-primary-foreground/70 mb-1 flex items-center gap-1 font-medium">
            <Bot size={10} />
            → {(message as any).source_agent_title || `Agent ${message.source_agent_id.slice(0, 8)}`}
          </div>
        )}
        <p className="whitespace-pre-wrap break-words">{message.content}</p>
        <div className={`flex items-center gap-1.5 mt-1 text-[10px] ${isUser ? 'text-primary-foreground/70' : 'text-muted-foreground'}`}>
          <span>{formatTime(message.created_at)}</span>
          {message.status !== 'executed' && message.status !== 'acknowledged' && (
            <span className="capitalize">({message.status})</span>
          )}
        </div>
      </div>
    </div>
  );
}

function ProjectUpdateItem({ update }: { update: ProjectUpdate }) {
  const iconMap: Record<string, React.ReactNode> = {
    info: <Info size={14} className="text-blue-500" />,
    milestone: <Milestone size={14} className="text-emerald-500" />,
    decision: <BrainCircuit size={14} className="text-purple-500" />,
    pm_decision: <BrainCircuit size={14} className="text-purple-500" />,
    error: <AlertTriangle size={14} className="text-destructive" />,
    agent_spawned: <Bot size={14} className="text-blue-500" />,
    agent_completed: <CheckCircle size={14} className="text-emerald-500" />,
  };

  const bgMap: Record<string, string> = {
    info: 'bg-blue-500/5 border-blue-500/10',
    milestone: 'bg-emerald-500/5 border-emerald-500/10',
    decision: 'bg-purple-500/5 border-purple-500/10',
    pm_decision: 'bg-purple-500/5 border-purple-500/10',
    error: 'bg-destructive/5 border-destructive/10',
    agent_spawned: 'bg-blue-500/5 border-blue-500/10',
    agent_completed: 'bg-emerald-500/5 border-emerald-500/10',
  };

  return (
    <div className={`flex items-start gap-2 py-1.5 px-3 rounded text-sm border ${bgMap[update.type] || 'bg-secondary/30 border-border'}`}>
      <span className="mt-0.5 shrink-0">{iconMap[update.type] || <Info size={14} className="text-muted-foreground" />}</span>
      <div className="flex-1 min-w-0">
        <span className="text-[10px] uppercase font-medium text-muted-foreground">{update.type.replace('_', ' ')}</span>
        <FormattedText text={parseContent(update.content)} className="break-words" />
      </div>
      <span className="text-[10px] text-muted-foreground shrink-0 mt-0.5">{formatTime(update.timestamp)}</span>
    </div>
  );
}

/** Merge updates and messages into a single sorted timeline */
export function buildTimeline(
  updates?: AgentUpdate[],
  messages?: AgentMessage[],
  projectUpdates?: ProjectUpdate[],
): TimelineEntry[] {
  const entries: TimelineEntry[] = [];

  if (updates) {
    for (const u of updates) {
      entries.push({ kind: 'update', data: u, timestamp: u.timestamp });
    }
  }

  if (messages) {
    for (const m of messages) {
      entries.push({ kind: 'message', data: m, timestamp: m.created_at });
    }
  }

  if (projectUpdates) {
    for (const p of projectUpdates) {
      entries.push({ kind: 'project_update', data: p, timestamp: p.timestamp });
    }
  }

  entries.sort((a, b) => parseDate(a.timestamp).getTime() - parseDate(b.timestamp).getTime());

  return entries;
}
