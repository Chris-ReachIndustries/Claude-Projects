import { useState, useEffect, useCallback } from 'react';
import { Folder, FolderOpen, ChevronRight, ChevronDown, X, Loader2 } from 'lucide-react';
import { fetchFolders, type FolderEntry } from '../api';

interface FolderPickerProps {
  isOpen: boolean;
  onClose: () => void;
  onSelect: (path: string) => void;
}

interface TreeNode {
  entry: FolderEntry;
  children: TreeNode[] | null; // null = not loaded
  expanded: boolean;
}

function FolderPicker({ isOpen, onClose, onSelect }: FolderPickerProps) {
  const [roots, setRoots] = useState<TreeNode[]>([]);
  const [selectedPath, setSelectedPath] = useState<string>('');
  const [loading, setLoading] = useState(false);
  const [loadingPath, setLoadingPath] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const loadFolder = useCallback(async (folderPath: string) => {
    const result = await fetchFolders(folderPath);
    return result.folders.map((f): TreeNode => ({
      entry: f,
      children: null,
      expanded: false,
    }));
  }, []);

  // Load root on open
  useEffect(() => {
    if (!isOpen) return;
    setLoading(true);
    setError(null);
    setSelectedPath('');
    loadFolder('')
      .then(setRoots)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [isOpen, loadFolder]);

  const toggleNode = async (path: string, nodes: TreeNode[], setNodes: (n: TreeNode[]) => void) => {
    const updated = await Promise.all(
      nodes.map(async (node) => {
        if (node.entry.path === path) {
          if (node.expanded) {
            return { ...node, expanded: false };
          }
          // Load children if not yet loaded
          if (node.children === null) {
            setLoadingPath(path);
            try {
              const children = await loadFolder(path);
              return { ...node, children, expanded: true };
            } finally {
              setLoadingPath(null);
            }
          }
          return { ...node, expanded: true };
        }
        if (node.children && node.expanded) {
          let updatedChildren = node.children;
          await toggleNode(path, node.children, (c) => {
            updatedChildren = c;
          });
          return { ...node, children: updatedChildren };
        }
        return node;
      }),
    );
    setNodes(updated);
  };

  const handleToggle = async (path: string) => {
    await toggleNode(path, roots, setRoots);
  };

  const renderTree = (nodes: TreeNode[], depth: number = 0): React.ReactNode => {
    return nodes.map((node) => {
      const isSelected = selectedPath === node.entry.path;
      const isLoading = loadingPath === node.entry.path;

      return (
        <div key={node.entry.path}>
          <div
            className={`flex items-center gap-1.5 py-1.5 px-2 rounded-md cursor-pointer transition-colors text-sm ${
              isSelected
                ? 'bg-accent-500/15 text-accent-300 border border-accent-500/25'
                : 'hover:bg-surface-raised text-white/70 border border-transparent'
            }`}
            style={{ paddingLeft: `${depth * 20 + 8}px` }}
            onClick={() => setSelectedPath(node.entry.path)}
            onDoubleClick={() => node.entry.hasChildren && handleToggle(node.entry.path)}
          >
            {/* Expand/collapse toggle */}
            <button
              className="w-4 h-4 flex items-center justify-center flex-shrink-0"
              onClick={(e) => {
                e.stopPropagation();
                if (node.entry.hasChildren) handleToggle(node.entry.path);
              }}
            >
              {isLoading ? (
                <Loader2 size={12} className="animate-spin text-white/40" />
              ) : node.entry.hasChildren ? (
                node.expanded ? (
                  <ChevronDown size={12} className="text-white/40" />
                ) : (
                  <ChevronRight size={12} className="text-white/40" />
                )
              ) : (
                <span className="w-3" />
              )}
            </button>

            {/* Folder icon */}
            {node.expanded ? (
              <FolderOpen size={14} className="text-accent-400 flex-shrink-0" />
            ) : (
              <Folder size={14} className="text-white/40 flex-shrink-0" />
            )}

            {/* Name */}
            <span className="truncate">{node.entry.name}</span>
          </div>

          {/* Children */}
          {node.expanded && node.children && node.children.length > 0 && (
            <div>{renderTree(node.children, depth + 1)}</div>
          )}
        </div>
      );
    });
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="bg-surface-base border border-white/[0.07] rounded-xl shadow-2xl w-full max-w-lg mx-4 flex flex-col max-h-[80vh]">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-white/[0.07]">
          <h2 className="text-lg font-semibold text-white/90">Select Working Directory</h2>
          <button
            onClick={onClose}
            className="p-1 text-white/40 hover:text-white/70 rounded transition-colors"
          >
            <X size={18} />
          </button>
        </div>

        {/* Selected path display */}
        <div className="px-5 py-2 bg-surface-overlay border-b border-white/[0.07]">
          <span className="text-xs text-white/40">Path: </span>
          <span className="text-xs text-white/70 font-mono">
            {selectedPath ? `~/${selectedPath}` : '~ (home)'}
          </span>
        </div>

        {/* Tree */}
        <div className="flex-1 overflow-y-auto px-3 py-3 min-h-[300px]">
          {loading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 size={24} className="animate-spin text-white/40" />
            </div>
          ) : error ? (
            <div className="text-center py-12">
              <p className="text-red-400 text-sm">{error}</p>
            </div>
          ) : roots.length === 0 ? (
            <div className="text-center py-12">
              <p className="text-white/40 text-sm">No folders found</p>
            </div>
          ) : (
            renderTree(roots)
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-3 px-5 py-4 border-t border-white/[0.07]">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm text-white/60 hover:text-white/80 transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={() => onSelect(selectedPath)}
            className="px-4 py-2 text-sm bg-accent-600 hover:bg-accent-500 text-white rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            disabled={!selectedPath}
          >
            Launch Agent Here
          </button>
        </div>
      </div>
    </div>
  );
}

export default FolderPicker;
