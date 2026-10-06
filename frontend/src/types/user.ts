import type { User } from './auth';

export type UserListItem = User;

export interface CreateUserRequest {
  email: string;
  password: string;
  full_name: string;
  student_code?: string;
  role_ids?: string[];
}

export interface UpdateProfileRequest {
  full_name?: string;
  student_code?: string;
  avatar_url?: string;
}

export interface ChangePasswordRequest {
  old_password: string;
  new_password: string;
}

export interface ResetPasswordRequest {
  new_password: string;
}

export interface ChangeRoleRequest {
  role_ids: string[];
}

export interface SearchUserParams {
  page?: number;
  limit?: number;
  search?: string;
  role_id?: string;
  is_deleted?: boolean;
}
