const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:9222';

function getApiKey(): string {
  if (typeof window === 'undefined') return '';
  return localStorage.getItem('cam_api_key') || '';
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_URL}/api${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${getApiKey()}`,
      ...(options?.headers || {}),
    },
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`API error ${res.status}: ${text}`);
  }
  return res.json();
}

// Many list endpoints return {data: [...]} — this unwraps to just the array
async function requestList<T>(path: string): Promise<T[]> {
  const result = await request<any>(path);
  if (Array.isArray(result)) return result;
  if (result && Array.isArray(result.data)) return result.data;
  return [];
}

// Agents
export async function fetchAgents() { return requestList<any>('/agents'); }
export async function fetchAgent(id: string) { return request<any>(`/agents/${id}`); }
export async function patchAgent(id: string, data: Record<string, unknown>) { return request<any>(`/agents/${id}`, { method: 'PATCH', body: JSON.stringify(data) }); }
export async function fetchAgentUpdates(id: string, limit = 100) { return requestList<any>(`/agents/${id}/updates?limit=${limit}`); }
export async function sendAgentUpdate(id: string, data: { type: string; content: string }) { return request(`/agents/${id}/updates`, { method: 'POST', body: JSON.stringify(data) }); }
export async function fetchAgentMessages(id: string) { return requestList<any>(`/agents/${id}/messages`); }
export async function sendMessage(id: string, content: string) { return request(`/agents/${id}/messages`, { method: 'POST', body: JSON.stringify({ content }) }); }
export async function closeAgent(id: string) { return request(`/agents/${id}/close`, { method: 'POST' }); }
export async function resumeAgent(id: string) { return request(`/agents/${id}/resume`, { method: 'POST' }); }

// Projects
export async function fetchProjects() { return requestList<any>('/projects'); }
export async function fetchProject(id: string) { return request<any>(`/projects/${id}`); }
export async function fetchProjectAgents(id: string) { return requestList<any>(`/projects/${id}/agents`); }
export async function fetchProjectUpdates(id: string, limit = 100) { return requestList<any>(`/projects/${id}/updates?limit=${limit}`); }
export async function startProject(id: string, prompt?: string) { return request(`/projects/${id}/start`, { method: 'POST', body: JSON.stringify({ initial_prompt: prompt || '' }) }); }
export async function pauseProject(id: string) { return request(`/projects/${id}/pause`, { method: 'POST' }); }
export async function completeProject(id: string) { return request(`/projects/${id}/complete`, { method: 'POST' }); }
export async function createProject(data: any) { return request('/projects', { method: 'POST', body: JSON.stringify(data) }); }
export async function spawnProjectAgent(id: string, data: any) { return request(`/projects/${id}/spawn-agent`, { method: 'POST', body: JSON.stringify(data) }); }
export async function fetchProjectFiles(id: string) { return requestList<any>(`/projects/${id}/files`); }

// Roles
export async function fetchRoles() { return requestList<any>('/roles'); }
export async function fetchRole(id: string) { return request<any>(`/roles/${id}`); }
export async function fetchRolesStats() { return request<any>('/roles/stats'); }

// Auth
export async function fetchApiKey() { return request<{apiKey: string}>('/auth/key'); }

// Settings
export async function fetchSettings() { return request<any>('/settings'); }
export async function patchSettings(data: Record<string, unknown>) { return request<any>('/settings', { method: 'PATCH', body: JSON.stringify(data) }); }
export async function fetchSetupStatus() { return request<any>('/settings/setup-status'); }
