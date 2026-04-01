// Backend stores UTC timestamps without 'Z' suffix — normalize for JS parsing
function parseUTC(date: string): number {
  const normalized = date.includes('Z') || date.includes('+') ? date : date.replace(' ', 'T') + 'Z';
  return new Date(normalized).getTime();
}

export function timeAgo(date: string): string {
  const now = Date.now();
  const then = parseUTC(date);
  const seconds = Math.floor((now - then) / 1000);

  if (seconds < 10) return 'just now';
  if (seconds < 60) return `${seconds}s ago`;

  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;

  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;

  const days = Math.floor(hours / 24);
  if (days < 30) return `${days}d ago`;

  const months = Math.floor(days / 30);
  if (months < 12) return `${months}mo ago`;

  const years = Math.floor(months / 12);
  return `${years}y ago`;
}

export function formatDate(date: string): string {
  const normalized = date.includes('Z') || date.includes('+') ? date : date.replace(' ', 'T') + 'Z';
  return new Date(normalized).toLocaleString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}
