import { apiClient } from '@/lib/axios';
import type {
  CommonResponse,
  PaginatedResponse,
  User,
  CreateUserRequest,
  UpdateProfileRequest,
  ChangeRoleRequest,
  ResetPasswordRequest,
  ChangePasswordRequest,
  SearchUserParams,
} from '@/types';

export const userService = {
  async searchUsers(params: SearchUserParams): Promise<PaginatedResponse<User>> {
    const res = await apiClient.get<PaginatedResponse<User>>('/users', { params });
    return res.data;
  },

  async getUserById(id: string): Promise<User> {
    const res = await apiClient.get<CommonResponse<User>>(`/users/${id}`);
    return res.data.data!;
  },

  async createUser(payload: CreateUserRequest): Promise<User> {
    const res = await apiClient.post<CommonResponse<User>>('/users', payload);
    return res.data.data!;
  },

  async updateUser(id: string, payload: UpdateProfileRequest): Promise<User> {
    const res = await apiClient.put<CommonResponse<User>>(`/users/${id}`, payload);
    return res.data.data!;
  },

  async changeRole(id: string, payload: ChangeRoleRequest): Promise<User> {
    const res = await apiClient.put<CommonResponse<User>>(`/users/${id}/roles`, payload);
    return res.data.data!;
  },

  async resetPassword(id: string, payload: ResetPasswordRequest): Promise<void> {
    await apiClient.post<CommonResponse<void>>(`/users/${id}/password`, payload);
  },

  async deleteUser(id: string): Promise<void> {
    await apiClient.delete<CommonResponse<void>>(`/users/${id}`);
  },

  async restoreUser(id: string): Promise<User> {
    const res = await apiClient.post<CommonResponse<User>>(`/users/${id}/restore`);
    return res.data.data!;
  },

  async revokeSessions(id: string): Promise<void> {
    await apiClient.post<CommonResponse<void>>(`/users/${id}/revoke-sessions`);
  },

  // Current User Operations
  async getCurrentUser(): Promise<User> {
    const res = await apiClient.get<CommonResponse<User>>('/users/current');
    return res.data.data!;
  },

  async updateCurrentProfile(payload: UpdateProfileRequest): Promise<User> {
    const res = await apiClient.put<CommonResponse<User>>('/users/current', payload);
    return res.data.data!;
  },

  async changeCurrentPassword(payload: ChangePasswordRequest): Promise<void> {
    await apiClient.post<CommonResponse<void>>('/users/current/password', payload);
  },

  async clearCurrentProfileAndMemories(): Promise<void> {
    await apiClient.delete<CommonResponse<void>>('/user/profile');
  },
};

