import { apiClient } from '@/lib/axios';
import type { CommonResponse } from '@/types';

export interface StudentProfileDTO {
  user_id: string;
  major: string;
  cohort: string;
  academic_standing: string;
  completed_credits: number;
  cumulative_gpa: number;
  target_gpa: number;
  advisor_notes: string;
  updated_at: string;
}

export interface UserMemoryDTO {
  id: string;
  user_id: string;
  category: string;
  memory_key: string;
  memory_value: string;
  confidence: number;
  source: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface StudentProfileBundleDTO {
  profile: StudentProfileDTO | null;
  memories: UserMemoryDTO[] | null;
}

export interface UpsertStudentProfilePayload {
  major: string;
  cohort: string;
  academic_standing: string;
  completed_credits: number;
  cumulative_gpa: number;
  target_gpa: number;
  advisor_notes: string;
}

export interface SaveUserMemoryPayload {
  category: string;
  memory_key: string;
  memory_value: string;
  confidence?: number;
}

export const memoryService = {
  async getMyBundle(): Promise<StudentProfileBundleDTO> {
    const res = await apiClient.get<CommonResponse<StudentProfileBundleDTO>>('/user/profile');
    return res.data.data ?? { profile: null, memories: [] };
  },

  async updateMyProfile(payload: UpsertStudentProfilePayload): Promise<StudentProfileDTO> {
    const res = await apiClient.put<CommonResponse<StudentProfileDTO>>('/user/profile', payload);
    return res.data.data!;
  },

  async listMyMemories(): Promise<UserMemoryDTO[]> {
    const res = await apiClient.get<CommonResponse<UserMemoryDTO[]>>('/user/memories');
    return res.data.data ?? [];
  },

  async saveMyMemory(payload: SaveUserMemoryPayload): Promise<UserMemoryDTO> {
    const res = await apiClient.post<CommonResponse<UserMemoryDTO>>('/user/memories', payload);
    return res.data.data!;
  },

  async deleteMyMemory(id: string): Promise<void> {
    await apiClient.delete<CommonResponse<void>>(`/user/memories/${id}`);
  },

  async getStudentBundle(userId: string): Promise<StudentProfileBundleDTO> {
    const res = await apiClient.get<CommonResponse<StudentProfileBundleDTO>>(`/admin/users/${userId}/memories`);
    return res.data.data ?? { profile: null, memories: [] };
  },
};
