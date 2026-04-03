'use client';

import { useEffect, useState, useCallback } from 'react';
import Link from 'next/link';
import { fetchProjects, createProject } from '@/lib/api';
import { useSSE } from '@/providers/sse-provider';
import type { Project } from '@/types';
import { StatusBadge } from '@/components/status-badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Plus, FolderKanban, X } from 'lucide-react';
import { timeAgo } from '@/lib/time';

export default function ProjectsPage() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const { subscribe } = useSSE();

  useEffect(() => {
    fetchProjects()
      .then(data => { setProjects(data || []); setLoading(false); })
      .catch(() => setLoading(false));
  }, []);

  useEffect(() => {
    const unsub1 = subscribe('project-created', (p: Project) => {
      setProjects(prev => [p, ...prev]);
    });
    const unsub2 = subscribe('project-updated', (p: Project) => {
      setProjects(prev => {
        const idx = prev.findIndex(x => x.id === p.id);
        if (idx >= 0) { const next = [...prev]; next[idx] = p; return next; }
        return [p, ...prev];
      });
    });
    const unsub3 = subscribe('project-deleted', (p: { id: string }) => {
      setProjects(prev => prev.filter(x => x.id !== p.id));
    });
    return () => { unsub1(); unsub2(); unsub3(); };
  }, [subscribe]);

  if (loading) return <div className="p-8 text-muted-foreground">Loading...</div>;

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-foreground">Projects</h1>
        <Button onClick={() => setShowForm(true)}>
          <Plus size={16} />
          New Project
        </Button>
      </div>

      {showForm && (
        <CreateProjectForm
          onClose={() => setShowForm(false)}
          onCreated={p => { setProjects(prev => [p, ...prev]); setShowForm(false); }}
        />
      )}

      {projects.length === 0 ? (
        <div className="text-center py-20 text-muted-foreground">
          <FolderKanban size={40} className="mx-auto mb-3 opacity-30" />
          <p>No projects yet. Create one to get started.</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {projects.map(project => (
            <Link key={project.id} href={`/projects/${project.id}`}>
              <Card className="hover:bg-accent/50 transition-colors cursor-pointer h-full">
                <CardContent className="p-5">
                  <div className="flex items-center justify-between mb-2">
                    <StatusBadge status={project.status} />
                    <span className="text-xs text-muted-foreground">{timeAgo(project.created_at)}</span>
                  </div>
                  <h3 className="font-semibold text-foreground mb-1 truncate">{project.name}</h3>
                  {project.description && (
                    <p className="text-sm text-muted-foreground line-clamp-2 mb-3">{project.description}</p>
                  )}
                  <div className="flex items-center gap-3 text-xs text-muted-foreground">
                    <span>Max agents: {project.max_concurrent}</span>
                    {project.pm_agent_id && <span>PM active</span>}
                  </div>
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}

function CreateProjectForm({ onClose, onCreated }: { onClose: () => void; onCreated: (p: Project) => void }) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [folderPath, setFolderPath] = useState('');
  const [maxConcurrent, setMaxConcurrent] = useState(3);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = useCallback(async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;
    setSaving(true);
    setError('');
    try {
      const p = await createProject({
        name: name.trim(),
        description: description.trim(),
        folder_path: folderPath.trim(),
        max_concurrent: maxConcurrent,
      });
      onCreated(p as Project);
    } catch (err: any) {
      setError(err.message || 'Failed to create project');
    } finally {
      setSaving(false);
    }
  }, [name, description, folderPath, maxConcurrent, onCreated]);

  return (
    <Card className="mb-6 animate-fade-in">
      <CardContent className="p-5">
        <div className="flex items-center justify-between mb-4">
          <h3 className="font-semibold text-foreground">New Project</h3>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground">
            <X size={16} />
          </button>
        </div>
        <form onSubmit={handleSubmit} className="space-y-3">
          <div>
            <label className="text-sm font-medium text-foreground block mb-1">Name *</label>
            <input
              type="text"
              value={name}
              onChange={e => setName(e.target.value)}
              placeholder="My Project"
              className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
              autoFocus
            />
          </div>
          <div>
            <label className="text-sm font-medium text-foreground block mb-1">Description</label>
            <textarea
              value={description}
              onChange={e => setDescription(e.target.value)}
              placeholder="What this project is about..."
              rows={2}
              className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring resize-none"
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="text-sm font-medium text-foreground block mb-1">Folder Path</label>
              <input
                type="text"
                value={folderPath}
                onChange={e => setFolderPath(e.target.value)}
                placeholder="/workspace/project"
                className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
              />
            </div>
            <div>
              <label className="text-sm font-medium text-foreground block mb-1">Max Concurrent Agents</label>
              <input
                type="number"
                value={maxConcurrent}
                onChange={e => setMaxConcurrent(Number(e.target.value))}
                min={1}
                max={20}
                className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
              />
            </div>
          </div>
          {error && <p className="text-sm text-destructive">{error}</p>}
          <div className="flex items-center justify-end gap-2 pt-2">
            <Button variant="outline" type="button" onClick={onClose}>Cancel</Button>
            <Button type="submit" disabled={saving || !name.trim()}>
              {saving ? 'Creating...' : 'Create Project'}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
