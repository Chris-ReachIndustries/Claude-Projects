'use client';

import { useState, useEffect, useMemo } from 'react';
import { fetchRoles, spawnProjectAgent } from '@/lib/api';
import type { Role } from '@/types';
import { Button } from '@/components/ui/button';
import { X, Search, ChevronRight, ChevronDown, Bot } from 'lucide-react';

interface SpawnAgentDialogProps {
  projectId: string;
  open: boolean;
  onClose: () => void;
  onSpawned?: () => void;
}

export function SpawnAgentDialog({ projectId, open, onClose, onSpawned }: SpawnAgentDialogProps) {
  const [roles, setRoles] = useState<Role[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [selectedRole, setSelectedRole] = useState<Role | null>(null);
  const [task, setTask] = useState('');
  const [image, setImage] = useState('');
  const [spawning, setSpawning] = useState(false);
  const [error, setError] = useState('');
  const [expandedCategories, setExpandedCategories] = useState<Set<string>>(new Set());

  useEffect(() => {
    if (open) {
      setLoading(true);
      fetchRoles()
        .then(data => { setRoles(data || []); setLoading(false); })
        .catch(() => setLoading(false));
    }
  }, [open]);

  const filtered = useMemo(() => {
    if (!search) return roles;
    const q = search.toLowerCase();
    return roles.filter(r =>
      r.name.toLowerCase().includes(q) ||
      r.category?.toLowerCase().includes(q) ||
      r.description?.toLowerCase().includes(q)
    );
  }, [roles, search]);

  const grouped = useMemo(() => {
    const map = new Map<string, Role[]>();
    for (const role of filtered) {
      const cat = role.category || 'Uncategorized';
      if (!map.has(cat)) map.set(cat, []);
      map.get(cat)!.push(role);
    }
    return Array.from(map.entries()).sort((a, b) => a[0].localeCompare(b[0]));
  }, [filtered]);

  const toggleCategory = (cat: string) => {
    setExpandedCategories(prev => {
      const next = new Set(prev);
      if (next.has(cat)) next.delete(cat);
      else next.add(cat);
      return next;
    });
  };

  const handleSpawn = async () => {
    if (!task.trim()) {
      setError('Task prompt is required');
      return;
    }
    setSpawning(true);
    setError('');
    try {
      await spawnProjectAgent(projectId, {
        role_id: selectedRole?.id || '',
        task: task.trim(),
        image: image.trim() || undefined,
      });
      setTask('');
      setSelectedRole(null);
      setImage('');
      onSpawned?.();
      onClose();
    } catch (err: any) {
      setError(err.message || 'Failed to spawn agent');
    } finally {
      setSpawning(false);
    }
  };

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/20" onClick={onClose} />
      <div className="relative bg-card border border-border rounded-xl shadow-lg w-full max-w-2xl max-h-[80vh] flex flex-col animate-fade-in">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-border">
          <div className="flex items-center gap-2">
            <Bot size={18} className="text-primary" />
            <h2 className="text-lg font-semibold text-foreground">Spawn Agent</h2>
          </div>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground">
            <X size={18} />
          </button>
        </div>

        <div className="flex-1 overflow-auto p-5 space-y-4">
          {/* Role selector */}
          <div>
            <label className="text-sm font-medium text-foreground mb-1.5 block">Role (optional)</label>
            {selectedRole ? (
              <div className="flex items-center gap-2 px-3 py-2 rounded-lg border border-primary/30 bg-primary/5">
                <Bot size={14} className="text-primary" />
                <div className="flex-1">
                  <span className="text-sm font-medium text-foreground">{selectedRole.name}</span>
                  <span className="text-xs text-muted-foreground ml-2">{selectedRole.category}</span>
                </div>
                <button onClick={() => setSelectedRole(null)} className="text-muted-foreground hover:text-foreground">
                  <X size={14} />
                </button>
              </div>
            ) : (
              <div>
                <div className="relative mb-2">
                  <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
                  <input
                    type="text"
                    value={search}
                    onChange={e => setSearch(e.target.value)}
                    placeholder="Search roles..."
                    className="w-full rounded-lg border border-input bg-background pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                  />
                </div>
                <div className="max-h-48 overflow-auto border border-border rounded-lg">
                  {loading ? (
                    <div className="p-4 text-center text-sm text-muted-foreground">Loading roles...</div>
                  ) : grouped.length === 0 ? (
                    <div className="p-4 text-center text-sm text-muted-foreground">No roles found</div>
                  ) : (
                    grouped.map(([cat, catRoles]) => (
                      <div key={cat}>
                        <button
                          onClick={() => toggleCategory(cat)}
                          className="flex items-center gap-1.5 w-full px-3 py-1.5 text-xs font-medium text-muted-foreground bg-secondary/40 hover:bg-secondary/60"
                        >
                          {expandedCategories.has(cat) ? <ChevronDown size={12} /> : <ChevronRight size={12} />}
                          {cat}
                          <span className="text-[10px]">({catRoles.length})</span>
                        </button>
                        {expandedCategories.has(cat) && catRoles.map(role => (
                          <button
                            key={role.id}
                            onClick={() => { setSelectedRole(role); setSearch(''); }}
                            className="flex items-start gap-2 w-full px-4 py-2 text-left hover:bg-accent text-sm border-t border-border/50"
                          >
                            <Bot size={12} className="mt-0.5 text-muted-foreground shrink-0" />
                            <div>
                              <div className="font-medium text-foreground">{role.name}</div>
                              {role.description && (
                                <div className="text-xs text-muted-foreground line-clamp-1">{role.description}</div>
                              )}
                            </div>
                          </button>
                        ))}
                      </div>
                    ))
                  )}
                </div>
              </div>
            )}
          </div>

          {/* Task */}
          <div>
            <label className="text-sm font-medium text-foreground mb-1.5 block">Task Prompt *</label>
            <textarea
              value={task}
              onChange={e => setTask(e.target.value)}
              rows={4}
              placeholder="Describe what this agent should do..."
              className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring resize-none"
            />
          </div>

          {/* Container image */}
          <div>
            <label className="text-sm font-medium text-foreground mb-1.5 block">Container Image (optional)</label>
            <input
              type="text"
              value={image}
              onChange={e => setImage(e.target.value)}
              placeholder="e.g. reach/god-tier (uses default if blank)"
              className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
            />
          </div>

          {error && (
            <p className="text-sm text-destructive">{error}</p>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-2 px-5 py-4 border-t border-border">
          <Button variant="outline" onClick={onClose}>Cancel</Button>
          <Button onClick={handleSpawn} disabled={spawning || !task.trim()}>
            {spawning ? 'Spawning...' : 'Spawn Agent'}
          </Button>
        </div>
      </div>
    </div>
  );
}
