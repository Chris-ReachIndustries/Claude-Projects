import { useState, useEffect } from 'react';
import { Users, Zap, MessageSquare, Activity } from 'lucide-react';
import { fetchAnalytics } from '../api';

interface AnalyticsData {
  totalAgents: number;
  activeNow: number;
  updatesToday: number;
  messagesToday: number;
}

function AnalyticsPanel() {
  const [data, setData] = useState<AnalyticsData | null>(null);

  useEffect(() => {
    fetchAnalytics().then(setData).catch(() => {});
  }, []);

  if (!data) return null;

  const stats = [
    { label: 'Agents', value: data.totalAgents, icon: Users, color: 'text-white/50' },
    { label: 'Active', value: data.activeNow, icon: Zap, color: data.activeNow > 0 ? 'text-emerald-400' : 'text-white/30' },
    { label: 'Updates', value: data.updatesToday, icon: Activity, color: 'text-white/50' },
    { label: 'Messages', value: data.messagesToday, icon: MessageSquare, color: 'text-white/50' },
  ];

  return (
    <div className="flex items-center gap-5 mt-1">
      {stats.map((s) => (
        <div key={s.label} className="flex items-center gap-1.5">
          <s.icon size={12} className="text-white/20" />
          <span className={`text-sm font-semibold tabular-nums ${s.color}`}>{s.value}</span>
          <span className="text-[11px] text-white/25">{s.label}</span>
        </div>
      ))}
    </div>
  );
}

export default AnalyticsPanel;
