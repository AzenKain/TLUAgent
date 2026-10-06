import { apiClient, CSRF_HEADER_NAME, getCsrfToken } from '@/lib/axios';

export interface ChatModelDTO {
  id: string;
  name: string;
  model_key: string;
  is_default: boolean;
  context_length: number;
  supports_vision: boolean;
  max_images: number;
  provider_name: string;
}

export interface ToolCallInfo {
  id?: string;
  name: string;
  status: 'running' | 'completed' | 'failed';
  title?: string;
  result?: string;
}

export interface ChatMessage {
  id: string;
  sender: 'user' | 'assistant';
  text: string;
  reasoning?: string;
  toolCalls?: ToolCallInfo[];
  images?: string[];
  sources?: string[];
  feedback?: 'up' | 'down';
  timestamp: string;
  canEscalate?: boolean;
}

export interface ChatResponse {
  session_id?: string;
  query: string;
  reply: string;
  sources: string[];
  timestamp: string;
  can_escalate?: boolean;
}

export interface ChatSessionDTO {
  id: string;
  user_id: string;
  title: string;
  model_id: string;
  created_at: string;
  updated_at: string;
}

export interface ChatMessageItemDTO {
  id: string;
  conversation_id: string;
  sender: 'user' | 'assistant';
  content: string;
  images?: string[];
  sources?: string[];
  feedback?: 'up' | 'down';
  created_at: string;
}

export interface ConversationDetailDTO {
  conversation: ChatSessionDTO;
  user?: {
    id: string;
    email: string;
    full_name: string;
    student_code: string;
    avatar_url?: string;
  };
  messages: ChatMessageItemDTO[];
}

export interface AdminConversationSummaryDTO {
  id: string;
  user_id: string;
  user_name: string;
  user_email: string;
  student_code: string;
  title: string;
  model_id: string;
  total_messages: number;
  thumbs_up: number;
  thumbs_down: number;
  created_at: string;
  updated_at: string;
}

export interface AdminChatStatsDTO {
  total_conversations: number;
  total_messages: number;
  feedback_up: number;
  feedback_down: number;
  satisfaction_rate: number;
}

export interface PaginatedResult<T> {
  status: boolean;
  message?: string;
  data: T;
  pagination?: {
    current_page: number;
    page_size: number;
    total_records: number;
    total_pages: number;
  };
}

export interface ChatStreamCallbacks {
  onChunk: (chunk: string) => void;
  onReasoning?: (chunk: string) => void;
  onToolCall?: (toolCall: ToolCallInfo) => void;
  onSources?: (sources: string[]) => void;
  onCanEscalate?: (canEscalate: boolean) => void;
  onDone?: (timestamp: string, sessionId?: string) => void;
  onError?: (error: Error) => void;
  onRehydrateRequired?: (sessionId?: string) => void;
}

export const chatService = {
  notifyGuestLeave(sessionId: string): void {
    if (!sessionId || typeof navigator === 'undefined' || !navigator.sendBeacon) return;
    try {
      const blob = new Blob([JSON.stringify({ session_id: sessionId })], { type: 'application/json' });
      navigator.sendBeacon('/api/chat/guest/leave', blob);
    } catch {}
  },

  async askAdvisor(query: string, modelId?: string, images?: string[], sessionId?: string): Promise<ChatResponse> {
    const res = await apiClient.post<ChatResponse>('/chat', {
      query,
      model_id: modelId,
      images,
      session_id: sessionId,
    });
    return res.data;
  },

  async askAdvisorStream(
    query: string,
    modelId?: string,
    images?: string[],
    history?: Array<{ role: string; content: string }>,
    sessionId?: string,
    callbacks?: ChatStreamCallbacks,
    signal?: AbortSignal
  ): Promise<void> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      [CSRF_HEADER_NAME]: getCsrfToken(),
    };

    const res = await fetch('/api/chat/stream', {
      method: 'POST',
      headers,
      signal,
      body: JSON.stringify({
        query,
        session_id: sessionId,
        model_id: modelId,
        images,
        history,
      }),
    });

    if (!res.ok) {
      throw new Error(`Chat stream failed with status ${res.status}`);
    }

    if (!res.body) {
      throw new Error('ReadableStream not supported on response body');
    }

    const reader = res.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split(/\r?\n/);
      buffer = lines.pop() || '';

      for (const line of lines) {
        const trimmed = line.trim();
        if (!trimmed || !trimmed.startsWith('data:')) continue;
        const jsonStr = trimmed.replace(/^data:\s*/, '');
        if (jsonStr === '[DONE]') continue;
        try {
          const payload = JSON.parse(jsonStr);
          if (payload.type === 'chunk' && payload.content) {
            callbacks?.onChunk(payload.content);
          } else if (payload.type === 'reasoning' && (payload.content || payload.reasoning_content)) {
            callbacks?.onReasoning?.(payload.content || payload.reasoning_content);
          } else if (payload.type === 'tool_call' && payload.tool_call) {
            callbacks?.onToolCall?.(payload.tool_call);
            if (payload.sources) {
              callbacks?.onSources?.(payload.sources);
            }
          } else if (payload.type === 'sources' && payload.sources) {
            callbacks?.onSources?.(payload.sources);
          } else if (payload.type === 'can_escalate') {
            callbacks?.onCanEscalate?.(true);
          } else if (payload.type === 'rehydrate_required') {
            callbacks?.onRehydrateRequired?.(payload.session_id);
          } else if (payload.type === 'done') {
            if (payload.sources) {
              callbacks?.onSources?.(payload.sources);
            }
            callbacks?.onDone?.(payload.timestamp || '', payload.session_id);
          } else if (payload.type === 'error') {
            callbacks?.onError?.(new Error(payload.message || 'Stream error'));
          }
        } catch {
        }
      }
    }
  },

  async getModels(): Promise<ChatModelDTO[]> {
    const res = await apiClient.get<{ status: boolean; data: ChatModelDTO[] }>('/chat/models');
    return res.data.data;
  },

  async listSessions(page = 1, limit = 20): Promise<PaginatedResult<ChatSessionDTO[]>> {
    const res = await apiClient.get<PaginatedResult<ChatSessionDTO[]>>('/chat/sessions', {
      params: { page, limit },
    });
    return res.data;
  },

  async createSession(title: string, modelId?: string): Promise<ChatSessionDTO> {
    const res = await apiClient.post<{ status: boolean; data: ChatSessionDTO }>('/chat/sessions', {
      title,
      model_id: modelId,
    });
    return res.data.data;
  },

  async getSession(sessionId: string): Promise<ConversationDetailDTO> {
    const res = await apiClient.get<{ status: boolean; data: ConversationDetailDTO }>(`/chat/sessions/${sessionId}`);
    return res.data.data;
  },

  async updateSessionTitle(sessionId: string, title: string): Promise<void> {
    await apiClient.patch(`/chat/sessions/${sessionId}`, { title });
  },

  async deleteSession(sessionId: string): Promise<void> {
    await apiClient.delete(`/chat/sessions/${sessionId}`);
  },

  async submitFeedback(messageId: string, feedback: 'up' | 'down' | 'clear'): Promise<void> {
    await apiClient.post(`/chat/messages/${messageId}/feedback`, { feedback });
  },

  async listAdminConversations(params: {
    q?: string;
    feedback?: string;
    model_id?: string;
    page?: number;
    limit?: number;
  }): Promise<PaginatedResult<AdminConversationSummaryDTO[]>> {
    const res = await apiClient.get<PaginatedResult<AdminConversationSummaryDTO[]>>('/admin/chats', {
      params,
    });
    return res.data;
  },

  async getAdminConversation(id: string): Promise<ConversationDetailDTO> {
    const res = await apiClient.get<{ status: boolean; data: ConversationDetailDTO }>(`/admin/chats/${id}`);
    return res.data.data;
  },

  async deleteAdminConversation(id: string): Promise<void> {
    await apiClient.delete(`/admin/chats/${id}`);
  },

  async getAdminStats(): Promise<AdminChatStatsDTO> {
    const res = await apiClient.get<{ status: boolean; data: AdminChatStatsDTO }>('/admin/chats/stats');
    return res.data.data;
  },
};
