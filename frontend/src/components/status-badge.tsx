'use client';

import { Badge } from '@/components/ui/badge';
import type { BadgeProps } from '@/components/ui/badge';

const statusVariantMap: Record<string, BadgeProps['variant']> = {
  active: 'active',
  working: 'working',
  idle: 'idle',
  'waiting-for-input': 'waiting',
  completed: 'secondary',
  archived: 'archived',
  pending: 'secondary',
  paused: 'waiting',
  failed: 'destructive',
};

export function StatusBadge({ status }: { status: string }) {
  const variant = statusVariantMap[status] || 'secondary';
  return (
    <Badge variant={variant} className="capitalize">
      {status.replace(/-/g, ' ')}
    </Badge>
  );
}
