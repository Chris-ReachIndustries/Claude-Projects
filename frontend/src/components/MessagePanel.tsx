import { useState, useRef } from 'react';
import { Send, Paperclip, Clock, CheckCircle, CheckCheck, PlayCircle, File as FileIcon, X, Bot, User } from 'lucide-react';
import type { AgentMessage } from '../types';
import { sendMessage, uploadFile } from '../api';
import { timeAgo } from '../utils/time';

const messageStatusConfig = {
  pending: { icon: Clock, color: 'text-yellow-400', bg: 'bg-yellow-400/10', label: 'Pending' },
  delivered: { icon: CheckCircle, color: 'text-blue-400', bg: 'bg-blue-400/10', label: 'Delivered' },
  acknowledged: { icon: CheckCheck, color: 'text-purple-400', bg: 'bg-purple-400/10', label: 'Acknowledged' },
  executed: { icon: PlayCircle, color: 'text-green-400', bg: 'bg-green-400/10', label: 'Executed' },
} as const;

interface MessagePanelProps {
  agentId: string;
  messages: AgentMessage[];
  onSent: () => void;
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes}B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)}MB`;
}

function MessagePanel({ agentId, messages, onSent }: MessagePanelProps) {
  const [input, setInput] = useState('');
  const [sending, setSending] = useState(false);
  const [sendError, setSendError] = useState<string | null>(null);
  const [attachedFile, setAttachedFile] = useState<File | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleSend = async () => {
    const content = input.trim();
    if ((!content && !attachedFile) || sending) return;

    try {
      setSending(true);
      setSendError(null);

      let messageContent = content;

      // Upload file first if attached
      if (attachedFile) {
        const result = await uploadFile(agentId, attachedFile);
        const fileRef = `[File attached: ${result.file.filename} (id=${result.file.id}, ${result.file.mimetype}, ${formatSize(result.file.size)}). Retrieve via GET /api/agents/${agentId}/files/${result.file.id}]`;
        messageContent = messageContent ? `${messageContent}\n\n${fileRef}` : fileRef;
      }

      if (messageContent) {
        await sendMessage(agentId, messageContent);
      }

      setInput('');
      setAttachedFile(null);
      onSent();
    } catch (err) {
      setSendError(err instanceof Error ? err.message : 'Failed to send');
    } finally {
      setSending(false);
    }
  };

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) setAttachedFile(file);
    // Reset input so the same file can be re-selected
    if (fileInputRef.current) fileInputRef.current.value = '';
  };

  return (
    <div className="bg-surface-base border border-white/[0.07] rounded-xl overflow-hidden flex flex-col">
      <div className="px-5 py-4 border-b border-white/[0.07]">
        <h2 className="text-sm font-semibold text-white/70 uppercase tracking-wide">
          Messages
        </h2>
      </div>

      {/* Input */}
      <div className="p-4 border-b border-white/[0.07]">
        <div className="flex gap-2">
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="Send a message to the agent..."
            rows={2}
            className="flex-1 bg-surface-overlay border border-white/[0.07] rounded-lg px-3 py-2 text-sm text-white/90 placeholder-white/30 resize-none focus:outline-none focus:ring-2 focus:ring-accent-500/20 focus:border-accent-500/40"
          />
          <div className="flex flex-col gap-1 self-end">
            <button
              onClick={() => fileInputRef.current?.click()}
              disabled={sending}
              className="px-3 py-2 bg-surface-raised hover:bg-surface-raised disabled:bg-surface-raised disabled:text-white/30 text-white/70 rounded-lg transition-colors"
              title="Attach file"
            >
              <Paperclip size={16} />
            </button>
            <button
              onClick={handleSend}
              disabled={(!input.trim() && !attachedFile) || sending}
              className="px-3 py-2 bg-accent-600 hover:bg-accent-500 disabled:bg-surface-raised disabled:text-white/40 text-white rounded-lg transition-colors"
            >
              <Send size={16} />
            </button>
          </div>
        </div>

        {/* Hidden file input */}
        <input
          ref={fileInputRef}
          type="file"
          onChange={handleFileSelect}
          className="hidden"
        />

        {/* Attached file indicator */}
        {attachedFile && (
          <div className="mt-2 flex items-center gap-2 px-2 py-1.5 bg-surface-overlay border border-white/[0.07] rounded-lg text-sm">
            <FileIcon size={14} className="text-accent-400 shrink-0" />
            <span className="text-white/80 truncate flex-1">{attachedFile.name}</span>
            <span className="text-white/40 text-xs shrink-0">{formatSize(attachedFile.size)}</span>
            <button
              onClick={() => setAttachedFile(null)}
              className="text-white/40 hover:text-white/80 transition-colors shrink-0"
            >
              <X size={14} />
            </button>
          </div>
        )}

        {sendError && (
          <p className="text-xs text-red-400 mt-2">{sendError}</p>
        )}
      </div>

      {/* Message list */}
      <div className="max-h-[calc(100vh-480px)] overflow-y-auto p-4 space-y-3">
        {messages.length === 0 ? (
          <p className="text-sm text-white/30 text-center py-4">No messages yet</p>
        ) : (
          [...messages].reverse().map((msg) => {
            const statusCfg = messageStatusConfig[msg.status];
            const StatusIcon = statusCfg.icon;
            const isAgent = msg.source === 'agent';

            return (
              <div
                key={msg.id}
                className={`p-3 rounded-lg border ${
                  isAgent
                    ? 'bg-blue-950/10 border-blue-800/20'
                    : msg.status === 'pending'
                      ? 'bg-yellow-950/10 border-yellow-800/20'
                      : 'bg-surface-overlay border-white/[0.07]/50'
                }`}
              >
                {/* Source label */}
                <div className="flex items-center gap-1.5 mb-1.5">
                  {isAgent ? (
                    <Bot size={13} className="text-blue-400" />
                  ) : (
                    <User size={13} className="text-purple-400" />
                  )}
                  <span className={`text-xs font-medium ${isAgent ? 'text-blue-400' : 'text-purple-400'}`}>
                    {isAgent
                      ? (msg.source_agent_id ? `Agent ${msg.source_agent_id}` : 'Agent')
                      : 'You'}
                  </span>
                </div>
                <p className="text-sm text-white/80 whitespace-pre-wrap break-words mb-2">
                  {msg.content}
                </p>
                <div className="flex items-center justify-between">
                  <span
                    className={`inline-flex items-center gap-1 text-xs ${statusCfg.color} ${statusCfg.bg} px-2 py-0.5 rounded-full`}
                  >
                    <StatusIcon size={10} />
                    {statusCfg.label}
                  </span>
                  <span className="text-xs text-white/30">
                    {timeAgo(msg.created_at)}
                  </span>
                </div>
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}

export default MessagePanel;
