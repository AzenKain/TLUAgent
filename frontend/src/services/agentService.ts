import { apiClient } from '@/lib/axios';
import type {
  AgentPromptDTO,
  UpdateAgentPromptDTO,
  AgentSkillDTO,
  CreateAgentSkillDTO,
  UpdateAgentSkillDTO,
  PreviewPromptDTO,
  CompactorSettingsDTO,
} from '@/types/agent';

export interface CommonResponse<T> {
  status: boolean;
  message?: string;
  data: T;
  errors?: unknown;
}

export const agentService = {
  async getPrompts(): Promise<AgentPromptDTO[]> {
    const res = await apiClient.get<CommonResponse<AgentPromptDTO[]>>('/admin/agent/prompts');
    return res.data.data;
  },

  async getPrompt(id: string): Promise<AgentPromptDTO> {
    const res = await apiClient.get<CommonResponse<AgentPromptDTO>>(`/admin/agent/prompts/${id}`);
    return res.data.data;
  },

  async updatePrompt(id: string, data: UpdateAgentPromptDTO): Promise<AgentPromptDTO> {
    const res = await apiClient.put<CommonResponse<AgentPromptDTO>>(`/admin/agent/prompts/${id}`, data);
    return res.data.data;
  },

  async resetPrompt(id: string): Promise<AgentPromptDTO> {
    const res = await apiClient.post<CommonResponse<AgentPromptDTO>>(`/admin/agent/prompts/${id}/reset`);
    return res.data.data;
  },

  async getSkills(): Promise<AgentSkillDTO[]> {
    const res = await apiClient.get<CommonResponse<AgentSkillDTO[]>>('/admin/agent/skills');
    return res.data.data;
  },

  async getSkill(id: string): Promise<AgentSkillDTO> {
    const res = await apiClient.get<CommonResponse<AgentSkillDTO>>(`/admin/agent/skills/${id}`);
    return res.data.data;
  },

  async createSkill(data: CreateAgentSkillDTO): Promise<AgentSkillDTO> {
    const res = await apiClient.post<CommonResponse<AgentSkillDTO>>('/admin/agent/skills', data);
    return res.data.data;
  },

  async updateSkill(id: string, data: UpdateAgentSkillDTO): Promise<AgentSkillDTO> {
    const res = await apiClient.put<CommonResponse<AgentSkillDTO>>(`/admin/agent/skills/${id}`, data);
    return res.data.data;
  },

  async toggleSkill(id: string, is_enabled: boolean): Promise<void> {
    await apiClient.patch(`/admin/agent/skills/${id}/toggle`, { is_enabled });
  },

  async deleteSkill(id: string): Promise<void> {
    await apiClient.delete(`/admin/agent/skills/${id}`);
  },

  async resetSkill(id: string): Promise<AgentSkillDTO> {
    const res = await apiClient.post<CommonResponse<AgentSkillDTO>>(`/admin/agent/skills/${id}/reset`);
    return res.data.data;
  },

  async previewPrompt(data: PreviewPromptDTO): Promise<string> {
    const res = await apiClient.post<CommonResponse<{ prompt: string }>>('/admin/agent/preview', data);
    return res.data.data.prompt;
  },

  async getCompactorSettings(): Promise<CompactorSettingsDTO> {
    const res = await apiClient.get<CommonResponse<CompactorSettingsDTO>>('/admin/agent/compactor');
    return res.data.data;
  },

  async updateCompactorSettings(data: CompactorSettingsDTO): Promise<CompactorSettingsDTO> {
    const res = await apiClient.put<CommonResponse<CompactorSettingsDTO>>('/admin/agent/compactor', data);
    return res.data.data;
  },
};
