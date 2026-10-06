import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { roleService } from '@/services/roleService';
import type { CreateRoleRequest, UpdateRoleRequest, PermissionAssignment } from '@/types';

export const ROLE_QUERY_KEYS = {
  all: ['roles'] as const,
  lists: () => [...ROLE_QUERY_KEYS.all, 'list'] as const,
  details: () => [...ROLE_QUERY_KEYS.all, 'detail'] as const,
  detail: (id: string) => [...ROLE_QUERY_KEYS.details(), id] as const,
  permissions: ['permissions', 'all'] as const,
  rolePermissions: (roleId: string) => ['roles', roleId, 'permissions'] as const,
};

export const useRoles = () => {
  const queryClient = useQueryClient();

  const rolesQuery = useQuery({
    queryKey: ROLE_QUERY_KEYS.lists(),
    queryFn: roleService.getRoles,
  });

  const permissionsQuery = useQuery({
    queryKey: ROLE_QUERY_KEYS.permissions,
    queryFn: roleService.getPermissions,
  });

  const createRoleMutation = useMutation({
    mutationFn: (data: CreateRoleRequest) => roleService.createRole(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ROLE_QUERY_KEYS.lists() });
    },
  });

  const updateRoleMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateRoleRequest }) =>
      roleService.updateRole(id, data),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: ROLE_QUERY_KEYS.lists() });
      queryClient.invalidateQueries({ queryKey: ROLE_QUERY_KEYS.detail(vars.id) });
    },
  });

  const deleteRoleMutation = useMutation({
    mutationFn: (id: string) => roleService.deleteRole(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ROLE_QUERY_KEYS.lists() });
    },
  });

  const updateRolePermissionsMutation = useMutation({
    mutationFn: ({ id, permissions }: { id: string; permissions: PermissionAssignment[] }) =>
      roleService.updateRolePermissions(id, permissions),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: ROLE_QUERY_KEYS.lists() });
      queryClient.invalidateQueries({ queryKey: ROLE_QUERY_KEYS.detail(vars.id) });
    },
  });

  const reorderRolesMutation = useMutation({
    mutationFn: (roleIds: string[]) => roleService.reorderRoles(roleIds),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ROLE_QUERY_KEYS.lists() });
    },
  });

  return {
    roles: rolesQuery.data || [],
    isLoadingRoles: rolesQuery.isLoading,
    isFetchingRoles: rolesQuery.isFetching,
    refetchRoles: rolesQuery.refetch,

    permissions: permissionsQuery.data || [],
    isLoadingPermissions: permissionsQuery.isLoading,
    isFetchingPermissions: permissionsQuery.isFetching,
    refetchPermissions: permissionsQuery.refetch,

    createRole: createRoleMutation.mutateAsync,
    isCreatingRole: createRoleMutation.isPending,

    updateRole: updateRoleMutation.mutateAsync,
    isUpdatingRole: updateRoleMutation.isPending,

    deleteRole: deleteRoleMutation.mutateAsync,
    isDeletingRole: deleteRoleMutation.isPending,

    updateRolePermissions: updateRolePermissionsMutation.mutateAsync,
    isUpdatingPermissions: updateRolePermissionsMutation.isPending,

    reorderRoles: reorderRolesMutation.mutateAsync,
    isReorderingRoles: reorderRolesMutation.isPending,
  };
};
