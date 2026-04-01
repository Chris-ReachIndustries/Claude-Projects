import { useMemo, useState } from 'react';
import type { Agent } from '../types';
import AgentCard from './AgentCard';
import FolderPicker from './FolderPicker';
import AnalyticsPanel from './AnalyticsPanel';
import { createLaunchRequest } from '../api';
import { RefreshCw, Bot, Archive, Plus, Search } from 'lucide-react';

type StatusFilter = 'all' | 'active' | 'idle' | 'working' | 'waiting-for-input' | 'completed' | 'archived';
type SortOption = 'activity' | 'created' | 'updates' | 'name';

const STATUS_CHIPS: { value: StatusFilter; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'active', label: 'Active' },
  { value: 'idle', label: 'Idle' },
  { value: 'working', label: 'Working' },
  { value: 'waiting-for-input', label: 'Waiting' },
  { value: 'completed', label: 'Completed' },
  { value: 'archived', label: 'Archived' },
];

const SORT_OPTIONS: { value: SortOption; label: string }[] = [
  { value: 'activity', label: 'Last Activity' },
  { value: 'created', label: 'Created' },
  { value: 'updates', label: 'Updates' },
  { value: 'name', label: 'Name A-Z' },
];

interface DashboardProps {
  agents: Agent[];
  loading: boolean;
  error: string | null;
  refetch: () => void;
}

function Dashboard({ agents, loading, error, refetch }: DashboardProps) {
  const [showFolderPicker, setShowFolderPicker] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [sortOption, setSortOption] = useState<SortOption>('activity');

  const handleLaunch = async (folderPath: string) => {
    try {
      await createLaunchRequest('new', folderPath);
      setShowFolderPicker(false);
    } catch (err) {
      console.error('Failed to create launch request:', err);
    }
  };

  const { activeAgents, archivedAgents } = useMemo(() => {
    // 1. Search filter
    const query = searchQuery.toLowerCase().trim();
    let filtered = agents;
    if (query) {
      filtered = agents.filter((a) => {
        const title = (a.title || '').toLowerCase();
        const workspace = (a.workspace || '').toLowerCase();
        const summary = (a.latest_summary || '').toLowerCase();
        return title.includes(query) || workspace.includes(query) || summary.includes(query);
      });
    }

    // 2. Status filter
    if (statusFilter !== 'all') {
      filtered = filtered.filter((a) => a.status === statusFilter);
    }

    // 3. Sort
    const sorted = [...filtered].sort((a, b) => {
      switch (sortOption) {
        case 'activity': {
          const aTime = a.last_activity_at || a.last_update_at;
          const bTime = b.last_activity_at || b.last_update_at;
          return new Date(bTime).getTime() - new Date(aTime).getTime();
        }
        case 'created':
          return new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
        case 'updates':
          return b.update_count - a.update_count;
        case 'name':
          return (a.title || '').localeCompare(b.title || '');
        default:
          return 0;
      }
    });

    // If user explicitly filters by archived, show them in the main section
    if (statusFilter === 'archived') {
      return {
        activeAgents: sorted,
        archivedAgents: [],
      };
    }

    return {
      activeAgents: sorted.filter((a) => a.status !== 'archived'),
      archivedAgents: sorted.filter((a) => a.status === 'archived'),
    };
  }, [agents, searchQuery, statusFilter, sortOption]);

  if (loading) {
    return (
      <div className="px-6 py-8">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {[...Array(6)].map((_, i) => (
            <div key={i} className="rounded-xl bg-surface-raised border border-white/[0.07] p-5 animate-pulse">
              <div className="flex items-center gap-2 mb-4">
                <div className="w-3 h-3 rounded-full bg-surface-overlay" />
                <div className="h-4 w-20 bg-surface-overlay rounded" />
              </div>
              <div className="h-5 w-3/4 bg-surface-overlay rounded mb-3" />
              <div className="h-4 w-full bg-surface-overlay rounded mb-2" />
              <div className="h-4 w-2/3 bg-surface-overlay rounded mb-4" />
              <div className="flex gap-4">
                <div className="h-3 w-12 bg-surface-overlay rounded" />
                <div className="h-3 w-12 bg-surface-overlay rounded" />
              </div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="px-6 py-8">
        <div className="rounded-xl bg-red-950/20 border border-red-500/20 p-6 text-center">
          <p className="text-red-400 mb-3">{error}</p>
          <button
            onClick={refetch}
            className="btn-secondary"
          >
            <RefreshCw size={14} />
            Retry
          </button>
        </div>
      </div>
    );
  }

  if (agents.length === 0) {
    return (
      <div className="px-6 py-8">
        <div className="flex flex-col items-center justify-center py-24 text-center">
          <div className="w-14 h-14 rounded-xl bg-surface-raised border border-white/[0.07] flex items-center justify-center mb-5">
            <Bot size={28} className="text-white/20" />
          </div>
          <h2 className="text-lg font-semibold text-white/60 mb-2">No agents yet</h2>
          <p className="text-white/30 text-sm max-w-md">
            Waiting for connections. Agents will appear here once they register with the manager.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="px-6 py-6 space-y-5">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-lg font-semibold text-white/90">Dashboard</h1>
          <AnalyticsPanel />
        </div>
        <button
          onClick={() => setShowFolderPicker(true)}
          className="btn-primary"
        >
          <Plus size={15} />
          New Agent
        </button>
      </div>

      {/* Search + filters + sort in one row */}
      <div className="flex items-center gap-3 flex-wrap">
        <div className="relative flex-1 max-w-sm">
          <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-white/20" />
          <input
            type="text"
            placeholder="Search agents..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="input-base pl-9"
          />
        </div>
        <div className="flex gap-1">
          {STATUS_CHIPS.map((chip) => (
            <button
              key={chip.value}
              onClick={() => setStatusFilter(chip.value)}
              className={`chip ${
                statusFilter === chip.value
                  ? 'bg-accent-500/15 text-accent-400 border-accent-500/30'
                  : 'bg-transparent text-white/30 border-white/[0.06] hover:text-white/50 hover:border-white/[0.12]'
              }`}
            >
              {chip.label}
            </button>
          ))}
        </div>

        <select
          value={sortOption}
          onChange={(e) => setSortOption(e.target.value as SortOption)}
          className="input-base w-auto"
        >
          {SORT_OPTIONS.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
      </div>

      <FolderPicker
        isOpen={showFolderPicker}
        onClose={() => setShowFolderPicker(false)}
        onSelect={handleLaunch}
      />

      {/* Active agents — 3-col grid */}
      {activeAgents.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {activeAgents.map((agent) => (
            <AgentCard key={agent.id} agent={agent} />
          ))}
        </div>
      )}

      {/* No results */}
      {activeAgents.length === 0 && archivedAgents.length === 0 && (searchQuery || statusFilter !== 'all') && (
        <div className="text-center py-12">
          <p className="text-white/30 text-sm">No agents match your filters.</p>
        </div>
      )}

      {/* Archived agents */}
      {archivedAgents.length > 0 && (
        <div>
          <div className="flex items-center gap-2 mb-4">
            <Archive size={14} className="text-white/25" />
            <h2 className="text-sm font-medium text-white/30 uppercase tracking-wide">
              Archived
            </h2>
            <span className="text-xs text-white/20">{archivedAgents.length}</span>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 opacity-60">
            {archivedAgents.map((agent) => (
              <AgentCard key={agent.id} agent={agent} />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

export default Dashboard;
