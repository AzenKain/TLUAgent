import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { userService } from '@/services/userService';
import type {
  SearchUserParams,
  CreateUserRequest,
  UpdateProfileRequest,
  ChangeRoleRequest,
  ResetPasswordRequest,
} from '@/types';

export const USER_QUERY_KEYS = {
  all: ['users'] as const,
  lists: () => [...USER_QUERY_KEYS.all, 'list'] as const,
  list: (params: SearchUserParams) => [...USER_QUERY_KEYS.lists(), params] as const,
  details: () => [...USER_QUERY_KEYS.all, 'detail'] as const,
  detail: (id: string) => [...USER_QUERY_KEYS.details(), id] as const,
  current: ['users', 'current'] as const,
};

export const useUsers = (params: SearchUserParams = {}) => {
  const queryClient = useQueryClient();

  const usersQuery = useQuery({
    queryKey: USER_QUERY_KEYS.list(params),
    queryFn: () => userService.searchUsers(params),
  });

  const createUserMutation = useMutation({
    mutationFn: (data: CreateUserRequest) => userService.createUser(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: USER_QUERY_KEYS.lists() });
    },
  });

  const updateUserMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateProfileRequest }) =>
      userService.updateUser(id, data),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: USER_QUERY_KEYS.lists() });
      queryClient.invalidateQueries({ queryKey: USER_QUERY_KEYS.detail(vars.id) });
    },
  });

  const changeRoleMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: ChangeRoleRequest }) =>
      userService.changeRole(id, data),
    onSuccess: (_, vars) => {
      queryClient.invalidateQueries({ queryKey: USER_QUERY_KEYS.lists() });
      queryClient.invalidateQueries({ queryKey: USER_QUERY_KEYS.detail(vars.id) });
    },
  });

  const resetPasswordMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: ResetPasswordRequest }) =>
      userService.resetPassword(id, data),
  });

  const deleteUserMutation = useMutation({
    mutationFn: (id: string) => userService.deleteUser(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: USER_QUERY_KEYS.lists() });
    },
  });

  const restoreUserMutation = useMutation({
    mutationFn: (id: string) => userService.restoreUser(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: USER_QUERY_KEYS.lists() });
    },
  });

  return {
    users: usersQuery.data?.data || [],
    pagination: usersQuery.data?.pagination,
    isLoading: usersQuery.isLoading,
    isError: usersQuery.isError,
    error: usersQuery.error,
    refetch: usersQuery.refetch,

    createUser: createUserMutation.mutateAsync,
    isCreating: createUserMutation.isPending,

    updateUser: updateUserMutation.mutateAsync,
    isUpdating: updateUserMutation.isPending,

    changeRole: changeRoleMutation.mutateAsync,
    isChangingRole: changeRoleMutation.isPending,

    resetPassword: resetPasswordMutation.mutateAsync,
    isResettingPassword: resetPasswordMutation.isPending,

    deleteUser: deleteUserMutation.mutateAsync,
    isDeleting: deleteUserMutation.isPending,

    restoreUser: restoreUserMutation.mutateAsync,
    isRestoring: restoreUserMutation.isPending,
  };
};

export const useUserDetail = (id: string) => {
  return useQuery({
    queryKey: USER_QUERY_KEYS.detail(id),
    queryFn: () => userService.getUserById(id),
    enabled: Boolean(id),
  });
};
