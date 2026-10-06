export interface Permission {
  key: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface PermissionAssignment {
  permission_key: string;
  effect: 'allow' | 'deny';
  conditions?: Record<string, unknown>;
}

export interface RolePermission {
  id: string;
  role_id: string;
  permission_key: string;
  effect: 'allow' | 'deny';
  conditions_json: string;
  conditions?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface Role {
  id: string;
  name: string;
  description: string;
  is_system: boolean;
  is_admin: boolean;
  is_banned: boolean;
  auto_assign: boolean;
  position: number;
  is_deleted: boolean;
  created_at: string;
  updated_at: string;
  permissions?: RolePermission[];
}

export interface CreateRoleRequest {
  name: string;
  description?: string;
  auto_assign?: boolean;
}

export interface UpdateRoleRequest {
  name?: string;
  description?: string;
  auto_assign?: boolean;
}

export interface UpdateRolePermissionsRequest {
  permissions: PermissionAssignment[];
}
