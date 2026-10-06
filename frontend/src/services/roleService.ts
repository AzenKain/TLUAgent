import { apiClient } from '@/lib/axios';
import type {
  CommonResponse,
  Role,
  Permission,
  RolePermission,
  CreateRoleRequest,
  UpdateRoleRequest,
  PermissionAssignment,
} from '@/types';

export const roleService = {
  async getRoles(): Promise<Role[]> {
    const res = await apiClient.get<CommonResponse<Role[]>>('/roles');
    return res.data.data!;
  },

  async getRoleById(id: string): Promise<Role> {
    const res = await apiClient.get<CommonResponse<Role>>(`/roles/${id}`);
    return res.data.data!;
  },

  async createRole(payload: CreateRoleRequest): Promise<Role> {
    const res = await apiClient.post<CommonResponse<Role>>('/roles', payload);
    return res.data.data!;
  },

  async updateRole(id: string, payload: UpdateRoleRequest): Promise<Role> {
    const res = await apiClient.put<CommonResponse<Role>>(`/roles/${id}`, payload);
    return res.data.data!;
  },

  async deleteRole(id: string): Promise<void> {
    await apiClient.delete<CommonResponse<void>>(`/roles/${id}`);
  },

  async getPermissions(): Promise<Permission[]> {
    const res = await apiClient.get<CommonResponse<Permission[]>>('/permissions');
    return res.data.data!;
  },

  async getRolePermissions(roleId: string): Promise<RolePermission[]> {
    const res = await apiClient.get<CommonResponse<RolePermission[]>>(`/roles/${roleId}/permissions`);
    return res.data.data!;
  },

  async updateRolePermissions(id: string, permissions: PermissionAssignment[]): Promise<Role> {
    const res = await apiClient.put<CommonResponse<Role>>(`/roles/${id}/permissions`, { permissions });
    return res.data.data!;
  },

  async reorderRoles(roleIds: string[]): Promise<void> {
    await apiClient.post<CommonResponse<void>>('/roles/reorder', { role_ids: roleIds });
  },
};
