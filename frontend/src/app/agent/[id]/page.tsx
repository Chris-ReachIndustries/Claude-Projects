'use client';

import { useEffect, useState, useCallback, use } from 'react';
import Link from 'next/link';
import {
  fetchAgent, fetchAgentUpdates, fetchAgentMessages,
  sendMessage, closeAgent, resumeAgent, patchAgent,
} from '@/lib/api';
import { useSSE } from '@/providers/sse-provider';
import type { Agent, AgentUpdate, AgentMessage, TimelineEntry } from '@/types';
import { UnifiedTimeline, buildTimeline } from '@/components/unified-timeline';
import { MessageInput } from '@/components/message-input';
import { StatusBadge } from '@/components/status-badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { ArrowLeft, Play, Archive, XCircle, Clock, Zap } from 'lucide-react';
import { timeAgo, formatDate } from '@/lib/time';

export default function AgentDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const [agent, setAgent] = useState<Agent | null>(null);
  const [updates, setUpdates] = useState<AgentUpdate[]>([]);
  const [messages, setMessages] = useState<AgentMessage[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const { subscribe } = useSSE();

  const load = useCallback(async () => {
    try {
      const [a, u, m] = await Promise.all([
        fetchAgent(id),
        fetchAgentUpdates(id, 500),
        fetchAgentMessages(id),
      ]) as [any, any[], any[]];
      setAgent(a);
      setUpdates(u || []);
      setMessages(m || []);
    } catch (err: any) {
      console.error('Failed to load agent:', err);
      setError(err?.message || 'Failed to load agent');
      setTimeout(() => setError(null), 5000);
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => { load(); }, [load]);

  // SSE subscriptions
  useEffect(() => {
    const unsub1 = subscribe('agent-updated', (a: Agent) => {
      if (a.id === id) setAgent(a);
    });
    const unsub2 = subscribe('message-queued', (m: AgentMessage) => {
      if (m.agent_id === id) {
        setMessages(prev => {
          if (prev.some(x => x.id === m.id)) return prev;
          return [...prev, m];
        });
      }
    });
    return () => { unsub1(); unsub2(); };
  }, [subscribe, id]);

  const handleSend = useCallback(async (content: string) => {
    await sendMessage(id, content);
    // Optimistically add message
    const optimistic: AgentMessage = {
      id: Date.now(),
      agent_id: id,
      content,
      role: 'user',
      status: 'pending',
      created_at: new Date().toISOString().replace('Z', ''),
    };
    setMessages(prev => [...prev, optimistic]);
  }, [id]);

  const handleClose = useCallback(async () => {
    try { await closeAgent(id); load(); } catch (err: any) { console.error('Failed to close agent:', err); setError(err?.message || 'Failed to close'); setTimeout(() => setError(null), 5000); }
  }, [id, load]);

  const handleResume = useCallback(async () => {
    try { await resumeAgent(id); load(); } catch (err: any) { console.error('Failed to resume agent:', err); setError(err?.message || 'Failed to resume'); setTimeout(() => setError(null), 5000); }
  }, [id, load]);

  const handleArchive = useCallback(async () => {
    try { await patchAgent(id, { status: 'archived' }); load(); } catch (err: any) { console.error('Failed to archive agent:', err); setError(err?.message || 'Failed to archive'); setTimeout(() => setError(null), 5000); }
  }, [id, load]);

  if (loading) return <div className="p-8 text-muted-foreground">Loading...</div>;
  if (!agent) return <div className="p-8 text-muted-foreground">Agent not found.</div>;

  const timeline: TimelineEntry[] = buildTimeline(updates, messages);

  const totalTokens = (agent.tokens_in || 0) + (agent.tokens_out || 0);
  const canSend = !['archived', 'completed'].includes(agent.status);

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="border-b border-border px-6 py-4 bg-card shrink-0">
        <div className="flex items-center gap-3 mb-1">
          <Link
            href={agent.project_id ? `/projects/${agent.project_id}` : '/'}
            className="text-muted-foreground hover:text-foreground"
          >
            <ArrowLeft size={18} />
          </Link>
          <h1 className="text-xl font-bold text-foreground">{agent.title || agent.role || 'Agent'}</h1>
          <StatusBadge status={agent.status} />
        </div>

        <div className="flex items-center gap-4 ml-8 text-xs text-muted-foreground flex-wrap">
          {agent.role && (
            <span className="text-primary font-medium">{agent.role}</span>
          )}
          <span className="flex items-center gap-1">
            <Clock size={12} />
            Created {formatDate(agent.created_at)}
          </span>
          {agent.last_activity_at && (
            <span>Active {timeAgo(agent.last_activity_at)}</span>
          )}
          {totalTokens > 0 && (
            <span className="flex items-center gap-1">
              <Zap size={12} />
              {totalTokens.toLocaleString()} tokens
            </span>
          )}
          <span>{agent.update_count} updates</span>
        </div>

        <div className="flex items-center gap-2 ml-8 mt-3">
          {['idle', 'waiting-for-input'].includes(agent.status) && (
            <Button size="sm" onClick={handleResume}><Play size={14} /> Resume</Button>
          )}
          {!['archived', 'completed'].includes(agent.status) && (
            <>
              <Button size="sm" variant="outline" onClick={handleArchive}>
                <Archive size={14} /> Archive
              </Button>
              <Button size="sm" variant="outline" onClick={handleClose}>
                <XCircle size={14} /> Close
              </Button>
            </>
          )}
        </div>
      </div>

      {/* Error banner */}
      {error && <div className="px-6 py-2 text-destructive text-sm bg-destructive/5 border-b border-destructive/10">{error}</div>}

      {/* Timeline */}
      <div className="flex-1 overflow-auto">
        <div className="max-w-4xl mx-auto">
          <UnifiedTimeline entries={timeline} className="py-3 px-2" />
        </div>
      </div>

      {/* Message input */}
      {canSend && (
        <MessageInput
          onSend={handleSend}
          placeholder="Send a message to this agent..."
        />
      )}
    </div>
  );
}
