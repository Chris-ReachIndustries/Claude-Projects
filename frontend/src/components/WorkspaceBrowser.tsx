import { useState, useEffect } from 'react';
import { Folder, FileText, ChevronRight, ArrowLeft, X } from 'lucide-react';
import { fetchWorkspaceFiles, fetchWorkspaceFile } from '../api';

interface FileEntry {
  name: string;
  path: string;
  is_dir: boolean;
  size: number;
  mod_time: string;
}

interface Props {
  rootPath?: string;
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes}B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)}MB`;
}

export default function WorkspaceBrowser({ rootPath = '.' }: Props) {
  const [currentPath, setCurrentPath] = useState(rootPath);
  const [files, setFiles] = useState<FileEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [previewFile, setPreviewFile] = useState<{ name: string; content: string } | null>(null);

  useEffect(() => {
    setLoading(true);
    setError(null);
    fetchWorkspaceFiles(currentPath)
      .then((data) => {
        setFiles(data.files || []);
      })
      .catch((err) => setError(err.message || 'Failed to load files'))
      .finally(() => setLoading(false));
  }, [currentPath]);

  const handleFileClick = async (file: FileEntry) => {
    if (file.is_dir) {
      setCurrentPath(file.path);
      return;
    }

    // Preview text files
    const ext = file.name.split('.').pop()?.toLowerCase() || '';
    const previewable = ['md', 'txt', 'py', 'js', 'ts', 'go', 'sh', 'json', 'yaml', 'yml', 'css', 'html', 'csv', 'sql', 'toml'];
    if (previewable.includes(ext)) {
      try {
        const content = await fetchWorkspaceFile(file.path);
        setPreviewFile({ name: file.name, content });
      } catch {
        // Can't preview
      }
    }
  };

  const navigateUp = () => {
    const parts = currentPath.split('/');
    parts.pop();
    setCurrentPath(parts.length > 0 ? parts.join('/') : '.');
  };

  const pathParts = currentPath === '.' ? [] : currentPath.split('/');

  return (
    <div className="space-y-3">
      {/* Breadcrumb */}
      <div className="flex items-center gap-1 text-xs text-white/40">
        <button onClick={() => setCurrentPath(rootPath)} className="hover:text-white/70 transition-colors">
          workspace
        </button>
        {pathParts.map((part, i) => (
          <span key={i} className="flex items-center gap-1">
            <ChevronRight size={10} className="text-white/20" />
            <button
              onClick={() => setCurrentPath(pathParts.slice(0, i + 1).join('/'))}
              className="hover:text-white/70 transition-colors"
            >
              {part}
            </button>
          </span>
        ))}
      </div>

      {/* File list */}
      {loading ? (
        <div className="py-8 text-center text-white/20 text-sm">Loading...</div>
      ) : error ? (
        <div className="py-8 text-center text-red-400/60 text-sm">{error}</div>
      ) : files.length === 0 ? (
        <div className="py-8 text-center text-white/20 text-sm">No files yet</div>
      ) : (
        <div className="border border-white/[0.06] rounded-lg overflow-hidden">
          {currentPath !== rootPath && (
            <button
              onClick={navigateUp}
              className="w-full flex items-center gap-3 px-4 py-2.5 text-sm text-white/40 hover:bg-white/[0.03] border-b border-white/[0.04] transition-colors"
            >
              <ArrowLeft size={14} />
              <span>..</span>
            </button>
          )}
          {files.map((file) => (
            <button
              key={file.path}
              onClick={() => handleFileClick(file)}
              className="w-full flex items-center gap-3 px-4 py-2.5 text-sm hover:bg-white/[0.03] border-b border-white/[0.04] last:border-b-0 transition-colors"
            >
              {file.is_dir ? (
                <Folder size={14} className="text-blue-400/60 shrink-0" />
              ) : (
                <FileText size={14} className="text-white/30 shrink-0" />
              )}
              <span className={`truncate ${file.is_dir ? 'text-blue-400/80' : 'text-white/60'}`}>
                {file.name}
              </span>
              {!file.is_dir && (
                <span className="ml-auto text-xs text-white/20 shrink-0">
                  {formatSize(file.size)}
                </span>
              )}
            </button>
          ))}
        </div>
      )}

      {/* Preview modal */}
      {previewFile && (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center p-8">
          <div className="bg-dark-950 border border-white/[0.08] rounded-xl max-w-4xl w-full max-h-[80vh] flex flex-col">
            <div className="flex items-center justify-between px-4 py-3 border-b border-white/[0.06]">
              <div className="flex items-center gap-2">
                <FileText size={14} className="text-white/40" />
                <span className="text-sm font-medium text-white/80">{previewFile.name}</span>
              </div>
              <button
                onClick={() => setPreviewFile(null)}
                className="p-1 rounded hover:bg-white/[0.06] text-white/40 hover:text-white/80 transition-colors"
              >
                <X size={16} />
              </button>
            </div>
            <div className="flex-1 overflow-auto p-4">
              <pre className="text-xs text-white/60 font-mono whitespace-pre-wrap leading-relaxed">
                {previewFile.content}
              </pre>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
