'use client';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { LayoutDashboard, FolderKanban, Settings } from 'lucide-react';
import { useSSE } from '@/providers/sse-provider';

const nav = [
  { href: '/', label: 'Dashboard', icon: LayoutDashboard },
  { href: '/projects', label: 'Projects', icon: FolderKanban },
  { href: '/settings', label: 'Settings', icon: Settings },
];

export function Sidebar() {
  const pathname = usePathname();
  const { state } = useSSE();

  return (
    <aside className="w-56 border-r border-border bg-secondary/30 flex flex-col">
      <div className="p-4 border-b border-border">
        <h1 className="text-lg font-bold text-foreground">Claude Projects</h1>
        <div className="flex items-center gap-1.5 mt-1">
          <span className={`w-1.5 h-1.5 rounded-full ${state === 'connected' ? 'bg-green-500' : state === 'connecting' ? 'bg-yellow-500 animate-pulse' : 'bg-red-500'}`} />
          <span className="text-xs text-muted-foreground capitalize">{state}</span>
        </div>
      </div>
      <nav className="flex-1 p-2">
        {nav.map(({ href, label, icon: Icon }) => {
          const active = pathname === href || (href !== '/' && pathname.startsWith(href));
          return (
            <Link
              key={href}
              href={href}
              className={`flex items-center gap-2 px-3 py-2 rounded-md text-sm transition-colors ${
                active ? 'bg-primary/10 text-primary font-medium' : 'text-muted-foreground hover:text-foreground hover:bg-accent'
              }`}
            >
              <Icon size={16} />
              {label}
            </Link>
          );
        })}
      </nav>
    </aside>
  );
}
