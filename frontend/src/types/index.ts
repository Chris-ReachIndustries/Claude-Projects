export interface Agent {
  id: string;
  title: string;
  status: 'active' | 'working' | 'idle' | 'waiting-for-input' | 'completed' | 'archived';
  created_at: string;
  last_update_at: string;
  last_activity_at: string | null;
  update_count: number;
  pending_message_count: number;
  latest_summary: string | null;
  cwd: string | null;
  project_id?: string;
  role?: string;
  parent_agent_id?: string;
  tokens_in?: number;
  tokens_out?: number;
}

export interface AgentUpdate {
  id: number;
  agent_id: string;
  type: 'text' | 'progress' | 'error' | 'status' | 'tool' | 'diagram' | 'thinking';
  content: string;
  metadata: string;
  timestamp: string;
}

export interface AgentMessage {
  id: number;
  agent_id: string;
  content: string;
  role: 'user' | 'assistant';
  status: 'pending' | 'delivered' | 'acknowledged' | 'executed';
  created_at: string;
  source_agent_id?: string;
}

export interface Project {
  id: string;
  name: string;
  description: string;
  status: 'pending' | 'active' | 'paused' | 'completed' | 'failed';
  folder_path: string;
  pm_agent_id: string | null;
  max_concurrent: number;
  created_at: string;
  started_at: string | null;
}

export interface ProjectUpdate {
  id: number;
  project_id: string;
  type: 'info' | 'milestone' | 'decision' | 'error' | 'agent_spawned' | 'agent_completed' | 'pm_decision';
  content: string;
  agent_id?: string;
  metadata?: string;
  timestamp: string;
}

export interface Role {
  id: string;
  name: string;
  category: string;
  description: string;
  system_prompt?: string;
}

export interface Settings {
  workspace_root: string;
  claude_config_path: string;
  default_image: string;
  agent_memory_limit: string;
  agent_cpu_limit: string;
}

export interface SetupStatus {
  complete: boolean;
  missing: string[];
}

export interface ProjectFile {
  name: string;
  size: number;
  agent_id?: string;
  agent_title?: string;
  modified_at: string;
}

export type TimelineEntry =
  | { kind: 'update'; data: AgentUpdate; timestamp: string }
  | { kind: 'message'; data: AgentMessage; timestamp: string }
  | { kind: 'project_update'; data: ProjectUpdate; timestamp: string };
