import { apiClient } from '@/lib/axios';

export interface LLMProviderDTO {
  id: string;
  name: string;
  provider_type: 'openai' | 'gemini' | 'openrouter' | 'custom';
  base_url: string;
  masked_api_key: string;
  is_active: boolean;
  is_default: boolean;
  custom_headers_json?: string;
  timeout_seconds?: number;
  max_retries?: number;
  retry_initial_wait_ms?: number;
  retry_max_wait_ms?: number;
  allow_private_networks?: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateLLMProviderDTO {
  id: string;
  name: string;
  provider_type: 'openai' | 'gemini' | 'openrouter' | 'custom';
  base_url: string;
  api_key: string;
  is_active?: boolean;
  is_default?: boolean;
  custom_headers_json?: string;
  timeout_seconds?: number;
  max_retries?: number;
  retry_initial_wait_ms?: number;
  retry_max_wait_ms?: number;
  allow_private_networks?: boolean;
}

export interface UpdateLLMProviderDTO {
  name: string;
  provider_type: 'openai' | 'gemini' | 'openrouter' | 'custom';
  base_url: string;
  api_key?: string;
  is_active?: boolean;
  is_default?: boolean;
  custom_headers_json?: string;
  timeout_seconds?: number;
  max_retries?: number;
  retry_initial_wait_ms?: number;
  retry_max_wait_ms?: number;
  allow_private_networks?: boolean;
}

export interface LLMModelDTO {
  id: string;
  provider_id: string;
  provider_name?: string;
  name: string;
  model_key: string;
  model_type: 'chat' | 'embedding' | 'rerank';
  is_active: boolean;
  is_default: boolean;
  order_index: number;
  context_length: number;
  vision_mode: 'auto' | 'manual';
  supports_vision: boolean;
  max_images: number;
  thinking_enabled?: boolean;
  thinking_budget?: number;
  effort_level?: 'low' | 'medium' | 'high';
  created_at: string;
  updated_at: string;
}

export interface CreateLLMModelDTO {
  id: string;
  provider_id: string;
  name: string;
  model_key: string;
  model_type: 'chat' | 'embedding' | 'rerank';
  is_active?: boolean;
  is_default?: boolean;
  order_index?: number;
  context_length?: number;
  vision_mode: 'auto' | 'manual';
  supports_vision?: boolean;
  max_images?: number;
  thinking_enabled?: boolean;
  thinking_budget?: number;
  effort_level?: 'low' | 'medium' | 'high';
}

export interface UpdateLLMModelDTO {
  provider_id: string;
  name: string;
  model_key: string;
  model_type: 'chat' | 'embedding' | 'rerank';
  is_active?: boolean;
  is_default?: boolean;
  order_index?: number;
  context_length?: number;
  vision_mode: 'auto' | 'manual';
  supports_vision?: boolean;
  max_images?: number;
  thinking_enabled?: boolean;
  thinking_budget?: number;
  effort_level?: 'low' | 'medium' | 'high';
}

export interface ProbeVisionDTO {
  provider_id: string;
  model_key: string;
}

export interface ProbeVisionResponseDTO {
  model_key: string;
  supports_vision: boolean;
  suggested_max_images: number;
}

export interface LLMChainNodeDTO {
  id: string;
  chain_id: string;
  model_id: string;
  priority: number;
  is_active: boolean;
  model_name: string;
  model_key: string;
  model_type: 'chat' | 'embedding' | 'rerank';
  context_length: number;
  supports_vision: boolean;
  max_images: number;
  provider_id: string;
  provider_name: string;
  provider_type: string;
  base_url: string;
  timeout_seconds: number;
  max_retries: number;
  health_state: 'closed' | 'open' | 'half_open';
  consecutive_failures: number;
  cooldown_until?: string;
  remaining_cooldown_seconds: number;
}

export interface LLMChainDTO {
  id: string;
  chain_type: 'chat' | 'rerank' | 'embedding';
  name: string;
  description: string;
  is_active: boolean;
  failure_threshold: number;
  cooldown_seconds: number;
  retry_count: number;
  nodes: LLMChainNodeDTO[];
  created_at: string;
  updated_at: string;
}

export interface CreateLLMChainDTO {
  id: string;
  chain_type: 'chat' | 'rerank' | 'embedding';
  name: string;
  description?: string;
  is_active?: boolean;
  failure_threshold: number;
  cooldown_seconds: number;
  retry_count: number;
}

export interface UpdateLLMChainDTO {
  name: string;
  description?: string;
  is_active?: boolean;
  failure_threshold: number;
  cooldown_seconds: number;
  retry_count: number;
}

export interface AddChainNodeDTO {
  model_id: string;
  priority: number;
  is_active?: boolean;
}

export interface UpdateChainNodePriorityDTO {
  priority: number;
  is_active?: boolean;
}

export const llmService = {
  async listProviders(): Promise<LLMProviderDTO[]> {
    const res = await apiClient.get<{ status: boolean; data: LLMProviderDTO[] }>('/admin/llm/providers');
    return res.data.data;
  },

  async getProvider(id: string): Promise<LLMProviderDTO> {
    const res = await apiClient.get<{ status: boolean; data: LLMProviderDTO }>(`/admin/llm/providers/${id}`);
    return res.data.data;
  },

  async createProvider(data: CreateLLMProviderDTO): Promise<LLMProviderDTO> {
    const res = await apiClient.post<{ status: boolean; data: LLMProviderDTO }>('/admin/llm/providers', data);
    return res.data.data;
  },

  async updateProvider(id: string, data: UpdateLLMProviderDTO): Promise<LLMProviderDTO> {
    const res = await apiClient.put<{ status: boolean; data: LLMProviderDTO }>(`/admin/llm/providers/${id}`, data);
    return res.data.data;
  },

  async deleteProvider(id: string): Promise<void> {
    await apiClient.delete(`/admin/llm/providers/${id}`);
  },

  async setDefaultProvider(id: string): Promise<void> {
    await apiClient.post(`/admin/llm/providers/${id}/default`);
  },

  async listModels(providerId?: string): Promise<LLMModelDTO[]> {
    const params = providerId ? { provider_id: providerId } : undefined;
    const res = await apiClient.get<{ status: boolean; data: LLMModelDTO[] }>('/admin/llm/models', { params });
    return res.data.data;
  },

  async createModel(data: CreateLLMModelDTO): Promise<LLMModelDTO> {
    const res = await apiClient.post<{ status: boolean; data: LLMModelDTO }>('/admin/llm/models', data);
    return res.data.data;
  },

  async updateModel(id: string, data: UpdateLLMModelDTO): Promise<LLMModelDTO> {
    const res = await apiClient.put<{ status: boolean; data: LLMModelDTO }>(`/admin/llm/models/${id}`, data);
    return res.data.data;
  },

  async deleteModel(id: string): Promise<void> {
    await apiClient.delete(`/admin/llm/models/${id}`);
  },

  async probeVision(data: ProbeVisionDTO): Promise<ProbeVisionResponseDTO> {
    const res = await apiClient.post<{ status: boolean; data: ProbeVisionResponseDTO }>('/admin/llm/models/probe-vision', data);
    return res.data.data;
  },

  async listChains(): Promise<LLMChainDTO[]> {
    const res = await apiClient.get<{ status: boolean; data: LLMChainDTO[] }>('/admin/llm/chains');
    return res.data.data;
  },

  async getChain(id: string): Promise<LLMChainDTO> {
    const res = await apiClient.get<{ status: boolean; data: LLMChainDTO }>(`/admin/llm/chains/${id}`);
    return res.data.data;
  },

  async createChain(data: CreateLLMChainDTO): Promise<LLMChainDTO> {
    const res = await apiClient.post<{ status: boolean; data: LLMChainDTO }>('/admin/llm/chains', data);
    return res.data.data;
  },

  async updateChain(id: string, data: UpdateLLMChainDTO): Promise<LLMChainDTO> {
    const res = await apiClient.put<{ status: boolean; data: LLMChainDTO }>(`/admin/llm/chains/${id}`, data);
    return res.data.data;
  },

  async deleteChain(id: string): Promise<void> {
    await apiClient.delete(`/admin/llm/chains/${id}`);
  },

  async addChainNode(chainId: string, data: AddChainNodeDTO): Promise<LLMChainNodeDTO> {
    const res = await apiClient.post<{ status: boolean; data: LLMChainNodeDTO }>(`/admin/llm/chains/${chainId}/nodes`, data);
    return res.data.data;
  },

  async removeChainNode(chainId: string, nodeId: string): Promise<void> {
    await apiClient.delete(`/admin/llm/chains/${chainId}/nodes/${nodeId}`);
  },

  async updateChainNodePriority(chainId: string, nodeId: string, data: UpdateChainNodePriorityDTO): Promise<void> {
    await apiClient.put(`/admin/llm/chains/${chainId}/nodes/${nodeId}`, data);
  },

  async resetNodeCircuitBreaker(chainId: string, nodeId: string): Promise<void> {
    await apiClient.post(`/admin/llm/chains/${chainId}/nodes/${nodeId}/reset`);
  },

  async testModel(data: TestModelRequestDTO): Promise<TestModelResponseDTO> {
    const res = await apiClient.post<{ status: boolean; data: TestModelResponseDTO }>('/admin/llm/test', data);
    return res.data.data;
  },
};

export interface TestModelRequestDTO {
  model_id?: string;
  chain_id?: string;
  system_prompt?: string;
  prompt: string;
  images?: string[];
  temperature?: number;
  max_tokens?: number;
}

export interface TestModelResponseDTO {
  success: boolean;
  model_id?: string;
  model_name?: string;
  model_key?: string;
  provider_name?: string;
  chain_id?: string;
  node_used?: string;
  content: string;
  reasoning_content?: string;
  latency_ms: number;
  prompt_tokens?: number;
  completion_tokens?: number;
  reasoning_tokens?: number;
  total_tokens?: number;
  finish_reason?: string;
  error?: string;
  raw_response?: Record<string, unknown>;
}
