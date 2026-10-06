import { apiClient } from '@/lib/axios';

export interface CommonResponse<T> {
  status: boolean;
  message?: string;
  data: T;
  errors?: unknown;
}

export interface InquiryDTO {
  id: string;
  user_id: string;
  conversation_id?: string;
  student_name: string;
  student_code: string;
  student_class?: string;
  question: string;
  context?: string;
  status: 'PENDING' | 'ANSWERED' | 'EXPIRED' | 'REJECTED';
  teacher_id?: string;
  teacher_name?: string;
  teacher_reply?: string;
  answered_at?: string;
  knowledge_chunk_id?: string;
  superseded_by_doc_id?: string;
  is_expired: boolean;
  expired_reason?: string;
  expired_at?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateInquiryPayload {
  student_name: string;
  student_code: string;
  student_class: string;
  question: string;
  context?: string;
  conversation_id?: string;
}

export interface NotificationDTO {
  id: string;
  user_id: string;
  inquiry_id?: string;
  title: string;
  content: string;
  type: string;
  is_read: boolean;
  created_at: string;
}

export const inquiryService = {
  async createInquiry(data: CreateInquiryPayload): Promise<InquiryDTO> {
    const res = await apiClient.post<CommonResponse<InquiryDTO>>('/inquiries', data);
    return res.data.data;
  },

  async getInquiry(id: string): Promise<InquiryDTO> {
    const res = await apiClient.get<CommonResponse<InquiryDTO>>(`/inquiries/${id}`);
    return res.data.data;
  },

  async getStudentInquiries(page = 1, limit = 20): Promise<{ items: InquiryDTO[]; total: number }> {
    const res = await apiClient.get<CommonResponse<{ items: InquiryDTO[]; total: number }>>('/student/inquiries', {
      params: { page, limit },
    });
    return res.data.data;
  },

  async getTeacherInquiries(status = '', search = '', page = 1, limit = 20): Promise<{ items: InquiryDTO[]; total: number }> {
    const res = await apiClient.get<CommonResponse<{ items: InquiryDTO[]; total: number }>>('/advisor/inquiries', {
      params: { status, search, page, limit },
    });
    return res.data.data;
  },

  async answerInquiry(id: string, reply: string): Promise<InquiryDTO> {
    const res = await apiClient.post<CommonResponse<InquiryDTO>>(`/advisor/inquiries/${id}/answer`, { reply });
    return res.data.data;
  },

  async expireInquiry(id: string, data: { superseded_by_doc_id?: string; reason: string }): Promise<void> {
    await apiClient.post(`/advisor/inquiries/${id}/expire`, data);
  },

  async getNotifications(page = 1, limit = 20): Promise<{ items: NotificationDTO[]; unread_count: number }> {
    const res = await apiClient.get<CommonResponse<{ items: NotificationDTO[]; unread_count: number }>>('/student/notifications', {
      params: { page, limit },
    });
    return res.data.data;
  },

  async markNotificationAsRead(id: string): Promise<void> {
    await apiClient.patch(`/student/notifications/${id}/read`);
  },

  async markAllNotificationsAsRead(): Promise<void> {
    await apiClient.post('/student/notifications/read-all');
  },

  async getUnreadCount(): Promise<number> {
    const res = await apiClient.get<CommonResponse<{ unread_count: number }>>('/student/notifications/unread-count');
    return res.data.data.unread_count;
  },
};
