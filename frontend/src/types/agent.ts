export interface AgentPromptDTO {
  id: string;
  title: string;
  content: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  updated_by?: string;
}

export interface UpdateAgentPromptDTO {
  title: string;
  content: string;
}

export interface AgentSkillDTO {
  id: string;
  name: string;
  description: string;
  content: string;
  tool_definition?: string;
  is_enabled: boolean;
  priority: number;
  created_at: string;
  updated_at: string;
  updated_by?: string;
}

export interface CreateAgentSkillDTO {
  id: string;
  name: string;
  description: string;
  content: string;
  tool_definition?: string;
  is_enabled?: boolean;
  priority?: number;
}

export interface UpdateAgentSkillDTO {
  name: string;
  description: string;
  content: string;
  tool_definition?: string;
  priority?: number;
}

export interface PreviewPromptDTO {
  student_cohort?: string;
  include_sample_rag?: boolean;
}

export interface CompactorSettingsDTO {
  max_context_tokens: number;
  compact_threshold_ratio: number;
  keep_recent_turns: number;
}

