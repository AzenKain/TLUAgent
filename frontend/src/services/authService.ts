import { apiClient } from '@/lib/axios';
import type {
  CommonResponse,
  AuthResponse,
  User,
  SignInRequest,
  SetupRequest,
  SetupStatusResponse,
} from '@/types';

export const authService = {
  async getSetupStatus(): Promise<SetupStatusResponse> {
    const res = await apiClient.get<CommonResponse<SetupStatusResponse>>('/setup/status');
    return res.data.data!;
  },

  async submitSetup(payload: SetupRequest): Promise<AuthResponse> {
    const res = await apiClient.post<CommonResponse<AuthResponse>>('/setup', payload);
    return res.data.data!;
  },

  async login(payload: SignInRequest): Promise<AuthResponse> {
    const res = await apiClient.post<CommonResponse<AuthResponse>>('/auth/login', payload);
    return res.data.data!;
  },

  async logout(): Promise<void> {
    await apiClient.post<CommonResponse<void>>('/auth/logout');
  },

  async getMe(): Promise<User> {
    const res = await apiClient.get<CommonResponse<User>>('/auth/me');
    return res.data.data!;
  },
};
