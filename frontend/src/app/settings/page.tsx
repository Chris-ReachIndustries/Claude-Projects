'use client';

import { useEffect, useState, useCallback } from 'react';
import { fetchSettings, patchSettings, fetchSetupStatus, fetchApiKey } from '@/lib/api';
import type { Settings, SetupStatus } from '@/types';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { AlertTriangle, CheckCircle, Key, Copy, RefreshCw } from 'lucide-react';

export default function SettingsPage() {
  const [settings, setSettings] = useState<Settings | null>(null);
  const [setupStatus, setSetupStatus] = useState<SetupStatus | null>(null);
  const [apiKey, setApiKey] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState('');
  const [showKey, setShowKey] = useState(false);
  const [copied, setCopied] = useState(false);

  // Form state
  const [workspaceRoot, setWorkspaceRoot] = useState('');
  const [claudeConfigPath, setClaudeConfigPath] = useState('');
  const [defaultImage, setDefaultImage] = useState('');
  const [memoryLimit, setMemoryLimit] = useState('');
  const [cpuLimit, setCpuLimit] = useState('');

  useEffect(() => {
    Promise.all([
      fetchSettings().catch(() => null),
      fetchSetupStatus().catch(() => null),
      fetchApiKey().catch(() => null),
    ]).then(([s, ss, ak]) => {
      if (s) {
        setSettings(s);
        setWorkspaceRoot(s.workspace_root || '');
        setClaudeConfigPath(s.claude_config_path || '');
        setDefaultImage(s.default_image || '');
        setMemoryLimit(s.agent_memory_limit || '');
        setCpuLimit(s.agent_cpu_limit || '');
      }
      if (ss) setSetupStatus(ss);
      if (ak) setApiKey(ak.apiKey || '');
      setLoading(false);
    });
  }, []);

  const handleSave = useCallback(async () => {
    setSaving(true);
    setError('');
    setSaved(false);
    try {
      const updated = await patchSettings({
        workspace_root: workspaceRoot,
        claude_config_path: claudeConfigPath,
        default_image: defaultImage,
        agent_memory_limit: memoryLimit,
        agent_cpu_limit: cpuLimit,
      });
      setSettings(updated);
      setSaved(true);
      setTimeout(() => setSaved(false), 3000);
    } catch (err: any) {
      setError(err.message || 'Failed to save settings');
    } finally {
      setSaving(false);
    }
  }, [workspaceRoot, claudeConfigPath, defaultImage, memoryLimit, cpuLimit]);

  const copyKey = useCallback(() => {
    navigator.clipboard.writeText(apiKey);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }, [apiKey]);

  if (loading) return <div className="p-8 text-muted-foreground">Loading...</div>;

  return (
    <div className="p-8 max-w-3xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold text-foreground">Settings</h1>

      {/* Setup status */}
      {setupStatus && !setupStatus.complete && (
        <Card className="border-orange-500/30 bg-orange-500/5">
          <CardContent className="p-5">
            <div className="flex items-start gap-3">
              <AlertTriangle size={18} className="text-orange-500 mt-0.5 shrink-0" />
              <div>
                <h3 className="font-semibold text-foreground mb-1">Setup Incomplete</h3>
                <p className="text-sm text-muted-foreground mb-2">
                  The following items need to be configured:
                </p>
                <ul className="space-y-1">
                  {setupStatus.missing?.map((item: string) => (
                    <li key={item} className="text-sm text-muted-foreground flex items-center gap-1.5">
                      <span className="w-1.5 h-1.5 rounded-full bg-orange-500" />
                      {item}
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {setupStatus?.complete && (
        <div className="flex items-center gap-2 text-sm text-emerald-600">
          <CheckCircle size={16} />
          Setup complete
        </div>
      )}

      {/* API Key */}
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base flex items-center gap-2">
            <Key size={16} />
            API Key
          </CardTitle>
          <CardDescription>Used for authenticating API requests and SSE connections.</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-2">
            <code className="flex-1 rounded-lg border border-input bg-secondary/30 px-3 py-2 text-sm font-mono text-foreground">
              {showKey ? apiKey : (apiKey ? '****' + apiKey.slice(-8) : 'No key available')}
            </code>
            <Button size="sm" variant="outline" onClick={() => setShowKey(!showKey)}>
              {showKey ? 'Hide' : 'Show'}
            </Button>
            {apiKey && (
              <Button size="sm" variant="outline" onClick={copyKey}>
                <Copy size={14} />
                {copied ? 'Copied' : 'Copy'}
              </Button>
            )}
          </div>
          <p className="text-xs text-muted-foreground mt-2">
            Store this key in localStorage as &quot;cam_api_key&quot; for automatic authentication.
          </p>
        </CardContent>
      </Card>

      {/* Settings form */}
      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Configuration</CardTitle>
          <CardDescription>Workspace and agent defaults.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <SettingsField
            label="Workspace Root"
            value={workspaceRoot}
            onChange={setWorkspaceRoot}
            placeholder="/home/user/workspace"
            help="Root directory for agent workspaces"
          />
          <SettingsField
            label="Claude Config Path"
            value={claudeConfigPath}
            onChange={setClaudeConfigPath}
            placeholder="/home/user/.claude/config.json"
            help="Path to Claude CLI configuration file"
          />
          <SettingsField
            label="Default Container Image"
            value={defaultImage}
            onChange={setDefaultImage}
            placeholder="reach/god-tier"
            help="Default Docker image for agent containers"
          />
          <div className="grid grid-cols-2 gap-4">
            <SettingsField
              label="Memory Limit"
              value={memoryLimit}
              onChange={setMemoryLimit}
              placeholder="4g"
              help="Docker memory limit per agent"
            />
            <SettingsField
              label="CPU Limit"
              value={cpuLimit}
              onChange={setCpuLimit}
              placeholder="2.0"
              help="Docker CPU limit per agent"
            />
          </div>

          {error && <p className="text-sm text-destructive">{error}</p>}

          <div className="flex items-center gap-3 pt-2">
            <Button onClick={handleSave} disabled={saving}>
              {saving ? 'Saving...' : 'Save Settings'}
            </Button>
            {saved && (
              <span className="text-sm text-emerald-600 flex items-center gap-1">
                <CheckCircle size={14} /> Saved
              </span>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function SettingsField({ label, value, onChange, placeholder, help }: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  help?: string;
}) {
  return (
    <div>
      <label className="text-sm font-medium text-foreground block mb-1">{label}</label>
      <input
        type="text"
        value={value}
        onChange={e => onChange(e.target.value)}
        placeholder={placeholder}
        className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
      />
      {help && <p className="text-xs text-muted-foreground mt-1">{help}</p>}
    </div>
  );
}
