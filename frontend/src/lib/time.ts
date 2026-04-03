export function parseDate(dateStr: string): Date {
  if (!dateStr) return new Date(0);
  // Handle both "2026-04-02 16:17:50" and "2026-04-02T16:17:50Z" formats
  const str = dateStr.trim();
  if (str.endsWith('Z') || str.includes('+')) {
    return new Date(str);
  }
  // Assume UTC if no timezone indicator
  return new Date(str + 'Z');
}

export function timeAgo(dateStr: string): string {
  if (!dateStr) return '';
  const date = parseDate(dateStr);
  if (isNaN(date.getTime())) return '';
  const now = new Date();
  const diff = Math.floor((now.getTime() - date.getTime()) / 1000);
  if (diff < 0) return 'just now';
  if (diff < 60) return `${diff}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return `${Math.floor(diff / 86400)}d ago`;
}

export function formatTime(dateStr: string): string {
  if (!dateStr) return '';
  const date = parseDate(dateStr);
  if (isNaN(date.getTime())) return '';
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

export function formatDate(dateStr: string): string {
  if (!dateStr) return '';
  const date = parseDate(dateStr);
  if (isNaN(date.getTime())) return '';
  return date.toLocaleString();
}
