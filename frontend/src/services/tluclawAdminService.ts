import { apiClient } from '@/lib/axios';
import type { CommonResponse } from '@/types/common';
import type {
  TLUClawSession,
  TLUClawChatRequest,
  TLUClawChatResponse,
  TLUClawConfirmRequest,
  TLUClawCancelRequest,
  TLUClawActionResponse,
  TLUClawSessionDetailResponse,
  TLUClawSnapshotItem,
} from '@/types/tluclaw';

export const tluclawAdminService = {
  async sendMessage(req: TLUClawChatRequest): Promise<TLUClawChatResponse> {
    const res = await apiClient.post<CommonResponse<TLUClawChatResponse>>('/admin/tluclaw/chat', req);
    return res.data.data!;
  },

  async confirmAction(req: TLUClawConfirmRequest): Promise<TLUClawActionResponse> {
    const res = await apiClient.post<CommonResponse<TLUClawActionResponse>>('/admin/tluclaw/confirm', req);
    return res.data.data!;
  },

  async cancelAction(req: TLUClawCancelRequest): Promise<TLUClawActionResponse> {
    const res = await apiClient.post<CommonResponse<TLUClawActionResponse>>('/admin/tluclaw/cancel', req);
    return res.data.data!;
  },

  async rollbackBatch(batchID: string): Promise<TLUClawActionResponse> {
    const res = await apiClient.post<CommonResponse<TLUClawActionResponse>>('/admin/tluclaw/rollback', {
      batch_id: batchID,
    });
    return res.data.data!;
  },

  async listSessions(limit = 20, offset = 0): Promise<TLUClawSession[]> {
    const res = await apiClient.get<CommonResponse<TLUClawSession[]>>('/admin/tluclaw/sessions', {
      params: { limit, offset },
    });
    return res.data.data || [];
  },

  async getSessionDetail(id: string): Promise<TLUClawSessionDetailResponse> {
    const res = await apiClient.get<CommonResponse<TLUClawSessionDetailResponse>>(`/admin/tluclaw/sessions/${id}`);
    return res.data.data!;
  },

  async createSession(title: string): Promise<TLUClawSession> {
    const res = await apiClient.post<CommonResponse<TLUClawSession>>('/admin/tluclaw/sessions', { title });
    return res.data.data!;
  },

  async deleteSession(id: string): Promise<void> {
    await apiClient.delete<CommonResponse<null>>(`/admin/tluclaw/sessions/${id}`);
  },

  async listSnapshots(status?: string, limit = 20, offset = 0): Promise<TLUClawSnapshotItem[]> {
    const res = await apiClient.get<CommonResponse<TLUClawSnapshotItem[]>>('/admin/tluclaw/snapshots', {
      params: { status: status || undefined, limit, offset },
    });
    return res.data.data || [];
  },
};
