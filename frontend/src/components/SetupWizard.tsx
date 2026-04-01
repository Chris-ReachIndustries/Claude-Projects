import { useState } from 'react';
import { FolderOpen, Settings as SettingsIcon, Check, ChevronRight, Loader2, Shield } from 'lucide-react';
import { updateSettings, setApiKey, fetchApiKey } from '../api';

interface Props {
  onComplete: () => void;
}

const AGENT_IMAGES = [
  { value: 'claude-agent', label: 'Standard', desc: 'Go CLI with 26 tools, bash, curl, git (~49MB)' },
];

export default function SetupWizard({ onComplete }: Props) {
  const [step, setStep] = useState(0);
  const [claudeConfigPath, setClaudeConfigPath] = useState('');
  const [workspaceRoot, setWorkspaceRoot] = useState('');
  const [defaultImage, setDefaultImage] = useState('claude-agent');
  const [memoryLimit, setMemoryLimit] = useState('2g');
  const [cpuLimit, setCpuLimit] = useState('1');
  const [maxAgents, setMaxAgents] = useState('8');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const steps = [
    { title: 'Welcome', icon: SettingsIcon },
    { title: 'Auth', icon: Shield },
    { title: 'Workspace', icon: FolderOpen },
    { title: 'Defaults', icon: SettingsIcon },
    { title: 'Complete', icon: Check },
  ];

  const canProceed = () => {
    if (step === 1) return claudeConfigPath.trim().length > 0;
    if (step === 2) return workspaceRoot.trim().length > 0;
    return true;
  };

  const handleFinish = async () => {
    setSaving(true);
    setError('');
    try {
      await updateSettings({
        claude_config_path: claudeConfigPath.trim(),
        workspace_root: workspaceRoot.trim(),
        default_agent_image: defaultImage,
        agent_memory_limit: memoryLimit,
        agent_cpu_limit: cpuLimit,
        max_concurrent_agents: maxAgents,
        setup_complete: 'true',
      });

      // Fetch and store the dashboard API key
      const dashKey = await fetchApiKey();
      setApiKey(dashKey);

      onComplete();
    } catch (err) {
      setError('Failed to save settings. Please try again.');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="min-h-screen bg-gray-950 flex items-center justify-center p-4">
      <div className="max-w-xl w-full bg-gray-900 rounded-2xl border border-gray-800 overflow-hidden">
        {/* Progress */}
        <div className="flex border-b border-gray-800">
          {steps.map((s, i) => (
            <div
              key={i}
              className={`flex-1 py-3 text-center text-xs font-medium transition-colors ${
                i === step ? 'bg-blue-600/20 text-blue-400 border-b-2 border-blue-500' :
                i < step ? 'text-green-400' : 'text-gray-600'
              }`}
            >
              {s.title}
            </div>
          ))}
        </div>

        <div className="p-8">
          {/* Step 0: Welcome */}
          {step === 0 && (
            <div className="text-center space-y-4">
              <h1 className="text-2xl font-bold text-white">Claude Agent Manager</h1>
              <p className="text-gray-400">
                Manage Claude Code agents running in Docker containers.
                Let's configure your environment.
              </p>
              <div className="text-sm text-gray-500 space-y-1">
                <p>Before starting, make sure you have:</p>
                <ul className="list-disc list-inside text-left mx-auto max-w-xs">
                  <li>Claude Code authenticated (VS Code extension or CLI)</li>
                  <li>A workspace directory for projects</li>
                  <li>Agent images built (<code className="text-gray-400">bash images/build.sh</code>)</li>
                </ul>
              </div>
            </div>
          )}

          {/* Step 1: Claude Auth */}
          {step === 1 && (
            <div className="space-y-4">
              <h2 className="text-xl font-semibold text-white">Claude Authentication</h2>
              <p className="text-sm text-gray-400">
                Agent containers need your Claude credentials to authenticate.
                This is the host path to your <code className="text-gray-300">~/.claude</code> directory
                (created by Claude Code VS Code extension or CLI).
              </p>
              <input
                type="text"
                value={claudeConfigPath}
                onChange={(e) => setClaudeConfigPath(e.target.value)}
                placeholder="/c/Users/yourname"
                className="w-full px-4 py-3 bg-gray-800 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-blue-500"
              />
              <p className="text-xs text-gray-500">
                Your home directory (Docker-style). On Windows: <code>/c/Users/yourname</code>. Must contain .claude/ and .claude.json
              </p>
            </div>
          )}

          {/* Step 2: Workspace */}
          {step === 2 && (
            <div className="space-y-4">
              <h2 className="text-xl font-semibold text-white">Workspace Root</h2>
              <p className="text-sm text-gray-400">
                The host directory where agent projects live. Each agent gets a subfolder
                mounted into its container.
              </p>
              <input
                type="text"
                value={workspaceRoot}
                onChange={(e) => setWorkspaceRoot(e.target.value)}
                placeholder="/c/Users/yourname/Projects"
                className="w-full px-4 py-3 bg-gray-800 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-blue-500"
              />
              <p className="text-xs text-gray-500">
                Docker-style path on Windows (e.g., <code>/c/Users/you/Projects</code>)
              </p>
            </div>
          )}

          {/* Step 3: Defaults */}
          {step === 3 && (
            <div className="space-y-4">
              <h2 className="text-xl font-semibold text-white">Agent Defaults</h2>

              <div>
                <label className="block text-sm text-gray-400 mb-1">Default Agent Image</label>
                <div className="space-y-2">
                  {AGENT_IMAGES.map((img) => (
                    <label
                      key={img.value}
                      className={`flex items-center gap-3 p-3 rounded-lg border cursor-pointer transition-colors ${
                        defaultImage === img.value
                          ? 'border-blue-500 bg-blue-500/10'
                          : 'border-gray-700 bg-gray-800 hover:border-gray-600'
                      }`}
                    >
                      <input
                        type="radio"
                        name="image"
                        value={img.value}
                        checked={defaultImage === img.value}
                        onChange={() => setDefaultImage(img.value)}
                        className="sr-only"
                      />
                      <div>
                        <span className="text-white font-medium">{img.label}</span>
                        <span className="text-gray-500 text-sm ml-2">{img.desc}</span>
                      </div>
                    </label>
                  ))}
                </div>
              </div>

              <div className="grid grid-cols-3 gap-3">
                <div>
                  <label className="block text-sm text-gray-400 mb-1">Memory Limit</label>
                  <input
                    type="text"
                    value={memoryLimit}
                    onChange={(e) => setMemoryLimit(e.target.value)}
                    className="w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded-lg text-white text-sm"
                  />
                </div>
                <div>
                  <label className="block text-sm text-gray-400 mb-1">CPU Limit</label>
                  <input
                    type="text"
                    value={cpuLimit}
                    onChange={(e) => setCpuLimit(e.target.value)}
                    className="w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded-lg text-white text-sm"
                  />
                </div>
                <div>
                  <label className="block text-sm text-gray-400 mb-1">Max Agents</label>
                  <input
                    type="text"
                    value={maxAgents}
                    onChange={(e) => setMaxAgents(e.target.value)}
                    className="w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded-lg text-white text-sm"
                  />
                </div>
              </div>
            </div>
          )}

          {/* Step 4: Complete */}
          {step === 4 && (
            <div className="text-center space-y-4">
              <div className="w-16 h-16 bg-green-500/20 rounded-full flex items-center justify-center mx-auto">
                <Check className="w-8 h-8 text-green-400" />
              </div>
              <h2 className="text-xl font-semibold text-white">Ready to Go</h2>
              <p className="text-gray-400">
                Your Agent Manager is configured. Click finish to start using the dashboard.
              </p>
              {error && (
                <p className="text-red-400 text-sm">{error}</p>
              )}
            </div>
          )}
        </div>

        {/* Navigation */}
        <div className="px-8 pb-8 flex justify-between">
          {step > 0 ? (
            <button
              onClick={() => setStep(step - 1)}
              className="px-4 py-2 text-gray-400 hover:text-white transition-colors"
            >
              Back
            </button>
          ) : <div />}

          {step < 4 ? (
            <button
              onClick={() => setStep(step + 1)}
              disabled={!canProceed()}
              className="flex items-center gap-2 px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              Next <ChevronRight size={16} />
            </button>
          ) : (
            <button
              onClick={handleFinish}
              disabled={saving}
              className="flex items-center gap-2 px-6 py-2 bg-green-600 text-white rounded-lg hover:bg-green-500 disabled:opacity-50 transition-colors"
            >
              {saving ? <Loader2 className="animate-spin" size={16} /> : <Check size={16} />}
              Finish
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
