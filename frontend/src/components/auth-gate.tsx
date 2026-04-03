'use client';

import { useState, useEffect } from 'react';

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:9222';

export function AuthGate({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<'checking' | 'login' | 'authenticated'>('checking');
  const [manualKey, setManualKey] = useState('');
  const [error, setError] = useState('');
  const [connecting, setConnecting] = useState(false);

  useEffect(() => {
    checkAuth();
  }, []);

  async function checkAuth() {
    setState('checking');
    const key = localStorage.getItem('cam_api_key');
    if (!key) {
      setState('login');
      return;
    }
    try {
      const res = await fetch(`${API_URL}/api/settings`, {
        headers: { Authorization: `Bearer ${key}` },
      });
      if (res.ok) {
        setState('authenticated');
      } else {
        localStorage.removeItem('cam_api_key');
        setState('login');
      }
    } catch {
      localStorage.removeItem('cam_api_key');
      setState('login');
    }
  }

  async function handleAutoConnect() {
    setError('');
    setConnecting(true);
    try {
      const res = await fetch(`${API_URL}/api/auth/key`);
      if (!res.ok) throw new Error('Failed');
      const data = await res.json();
      if (data.apiKey) {
        localStorage.setItem('cam_api_key', data.apiKey);
        setState('authenticated');
        // Force SSE reconnect with new key
        window.location.reload();
      }
    } catch {
      setError(`Could not connect to ${API_URL}. Is the backend running?`);
    } finally {
      setConnecting(false);
    }
  }

  function handleManualKey() {
    if (!manualKey.trim()) return;
    localStorage.setItem('cam_api_key', manualKey.trim());
    window.location.reload();
  }

  // Never render app content unless fully authenticated
  if (state === 'checking') {
    return (
      <div className="h-screen flex items-center justify-center bg-white">
        <div className="animate-spin w-6 h-6 border-2 border-violet-500 border-t-transparent rounded-full" />
      </div>
    );
  }

  if (state === 'login') {
    return (
      <div className="h-screen flex items-center justify-center bg-white">
        <div className="w-full max-w-sm mx-auto p-8">
          <h1 className="text-2xl font-bold text-gray-900 mb-1">Claude Projects</h1>
          <p className="text-sm text-gray-500 mb-8">Connect to your dashboard</p>

          <button
            onClick={handleAutoConnect}
            disabled={connecting}
            className="w-full px-4 py-3 rounded-lg bg-violet-600 text-white font-medium hover:bg-violet-700 transition-colors mb-4 disabled:opacity-50"
          >
            {connecting ? 'Connecting...' : 'Connect Automatically'}
          </button>

          <div className="relative my-6">
            <div className="absolute inset-0 flex items-center">
              <div className="w-full border-t border-gray-200" />
            </div>
            <div className="relative flex justify-center text-xs">
              <span className="px-2 bg-white text-gray-400">or enter key manually</span>
            </div>
          </div>

          <div className="flex gap-2">
            <input
              type="text"
              value={manualKey}
              onChange={e => setManualKey(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && handleManualKey()}
              placeholder="API key"
              className="flex-1 px-3 py-2 text-sm rounded-md border border-gray-200 bg-gray-50 text-gray-900 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-violet-500"
            />
            <button
              onClick={handleManualKey}
              className="px-4 py-2 text-sm rounded-md bg-gray-100 text-gray-700 border border-gray-200 hover:bg-gray-200 transition-colors"
            >
              Connect
            </button>
          </div>

          {error && <p className="mt-4 text-xs text-red-500">{error}</p>}
        </div>
      </div>
    );
  }

  return <>{children}</>;
}
