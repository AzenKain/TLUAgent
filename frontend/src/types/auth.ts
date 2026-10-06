import type { RolePermission } from './role';

export interface RoleSimple {
  id: string;
  name: string;
  is_admin: boolean;
  is_banned: boolean;
  position: number;
  permissions?: RolePermission[];
}

export interface User {
  id: string;
  email: string;
  full_name: string;
  student_code?: string;
  avatar_url?: string;
  auth_provider: string;
  token_version: number;
  roles: RoleSimple[];
  is_owner?: boolean;
  is_deleted?: boolean;
  created_at: string;
  updated_at: string;
}

export interface AuthResponse {
  user: User;
}

export interface SetupStatusResponse {
  required: boolean;
  message: string;
}

export interface SignInRequest {
  email: string;
  password: string;
}

export interface SetupRequest {
  email: string;
  password: string;
  full_name: string;
  student_code?: string;
}
