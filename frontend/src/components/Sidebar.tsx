import { memo, useState } from 'react';
import { NavLink, useLocation } from 'react-router-dom';
import {
  LayoutDashboard, FolderKanban, GitBranch, Settings, ChevronLeft, ChevronRight, Zap,
} from 'lucide-react';
import type { ConnectionState } from '../api';

interface SidebarProps {
  agentCount: number;
  activeCount: number;
  connectionState: ConnectionState;
}

const connectionConfig: Record<ConnectionState, { color: string; pulse: boolean; label: string }> = {
  connected: { color: 'bg-emerald-400', pulse: false, label: 'Connected' },
  connecting: { color: 'bg-amber-400', pulse: true, label: 'Reconnecting' },
  disconnected: { color: 'bg-red-400', pulse: false, label: 'Disconnected' },
};

function Sidebar({ agentCount, activeCount, connectionState }: SidebarProps) {
  const [collapsed, setCollapsed] = useState(false);
  const location = useLocation();
  const conn = connectionConfig[connectionState];

  const navItems = [
    { to: '/', icon: LayoutDashboard, label: 'Dashboard', exact: true },
    { to: '/projects', icon: FolderKanban, label: 'Projects' },
    { to: '/workflows', icon: GitBranch, label: 'Workflows' },
    { to: '/settings', icon: Settings, label: 'Settings' },
  ];

  return (
    <aside className={`fixed top-0 left-0 h-full z-50 flex flex-col bg-surface-base border-r border-white/[0.07] transition-all duration-200 ${collapsed ? 'w-14' : 'w-52'}`}>
      {/* Logo */}
      <NavLink to="/" className="flex items-center gap-2.5 px-3.5 h-14 border-b border-white/[0.07] hover:bg-white/[0.03] transition-colors">
        <div className="w-7 h-7 rounded-lg bg-gradient-to-br from-violet-500 to-fuchsia-600 flex items-center justify-center shrink-0">
          <Zap size={14} className="text-white" />
        </div>
        {!collapsed && (
          <span className="text-[13px] font-semibold text-white/90 tracking-tight">Claude Projects</span>
        )}
      </NavLink>

      {/* Nav */}
      <nav className="flex-1 py-2 px-2 space-y-0.5">
        {navItems.map((item) => {
          const isActive = item.exact
            ? location.pathname === item.to
            : location.pathname.startsWith(item.to);

          return (
            <NavLink
              key={item.to}
              to={item.to}
              className={`relative flex items-center gap-2.5 px-2.5 py-2 rounded-lg text-[13px] font-medium transition-all duration-100 ${
                isActive
                  ? 'bg-white/[0.08] text-white'
                  : 'text-white/40 hover:text-white/60 hover:bg-white/[0.04]'
              }`}
            >
              {isActive && (
                <div className="absolute left-0 top-1.5 bottom-1.5 w-[3px] rounded-r-full bg-accent-500" />
              )}
              <item.icon size={16} strokeWidth={isActive ? 2 : 1.5} className={isActive ? 'text-accent-400' : ''} />
              {!collapsed && <span>{item.label}</span>}
            </NavLink>
          );
        })}
      </nav>

      {/* Footer */}
      <div className={`px-3 py-3 border-t border-white/[0.07] space-y-2 ${collapsed ? 'px-1.5' : ''}`}>
        {!collapsed ? (
          <>
            <div className="flex items-center justify-between text-[11px]">
              <span className="text-white/30">{agentCount} agent{agentCount !== 1 ? 's' : ''}</span>
              {activeCount > 0 && (
                <span className="flex items-center gap-1.5">
                  <span className="w-1.5 h-1.5 rounded-full bg-emerald-400" />
                  <span className="text-white/50 font-medium">{activeCount} active</span>
                </span>
              )}
              {activeCount === 0 && (
                <span className="flex items-center gap-1.5">
                  <span className="w-1.5 h-1.5 rounded-full bg-red-400" />
                  <span className="text-white/30">none active</span>
                </span>
              )}
            </div>
            <div className="flex items-center gap-1.5">
              <div className={`w-1.5 h-1.5 rounded-full ${conn.color} ${conn.pulse ? 'animate-pulse' : ''}`} />
              <span className="text-[11px] text-white/25">{conn.label}</span>
            </div>
          </>
        ) : (
          <div className="flex flex-col items-center gap-1.5">
            <div className={`w-1.5 h-1.5 rounded-full ${conn.color}`} />
            <span className="text-[9px] text-white/20">{activeCount}</span>
          </div>
        )}
      </div>

      {/* Collapse */}
      <button
        onClick={() => setCollapsed(!collapsed)}
        className="flex items-center justify-center h-9 border-t border-white/[0.07] text-white/20 hover:text-white/50 hover:bg-white/[0.03] transition-colors"
      >
        {collapsed ? <ChevronRight size={14} /> : <ChevronLeft size={14} />}
      </button>
    </aside>
  );
}

export default memo(Sidebar);
