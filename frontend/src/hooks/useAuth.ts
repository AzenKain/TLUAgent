import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { isAxiosError } from 'axios';
import { authService } from '@/services/authService';
import { useAuthStore } from '@/stores/authStore';
import { useShallow } from 'zustand/react/shallow';
import type { SignInRequest, SetupRequest } from '@/types';

export const useAuth = () => {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const { user, isAuthenticated, setAuth, logout: storeLogout, setUser } = useAuthStore(
    useShallow((s) => ({
      user: s.user,
      isAuthenticated: s.isAuthenticated,
      setAuth: s.setAuth,
      logout: s.logout,
      setUser: s.setUser,
    }))
  );

  const setupStatusQuery = useQuery({
    queryKey: ['setup-status'],
    queryFn: async () => {
      try {
        return await authService.getSetupStatus();
      } catch (error) {
        if (isAxiosError(error) && error.response?.status === 404) {
          return { required: false, message: 'Setup already completed' };
        }
        throw error;
      }
    },
    enabled: !isAuthenticated,
    initialData: isAuthenticated ? { required: false, message: 'Setup completed' } : undefined,
    staleTime: Infinity,
    retry: false,
  });

  const meQuery = useQuery({
    queryKey: ['auth-me'],
    queryFn: async () => {
      const u = await authService.getMe();
      setUser(u);
      return u;
    },
    enabled: isAuthenticated,
    staleTime: 1000 * 60 * 5,
  });

  const loginMutation = useMutation({
    mutationFn: (data: SignInRequest) => authService.login(data),
    onSuccess: (data) => {
      setAuth(data.user);
      queryClient.invalidateQueries({ queryKey: ['auth-me'] });
      navigate('/');
    },
  });

  const setupMutation = useMutation({
    mutationFn: (data: SetupRequest) => authService.submitSetup(data),
    onSuccess: (data) => {
      setAuth(data.user);
      queryClient.invalidateQueries({ queryKey: ['setup-status'] });
      queryClient.invalidateQueries({ queryKey: ['auth-me'] });
      navigate('/');
    },
  });

  const logoutMutation = useMutation({
    mutationFn: authService.logout,
    onSettled: () => {
      storeLogout();
      queryClient.clear();
      navigate('/login');
    },
  });

  return {
    user,
    isAuthenticated,
    setupStatus: setupStatusQuery.data,
    isSetupLoading: setupStatusQuery.isLoading,
    login: loginMutation.mutateAsync,
    isLoggingIn: loginMutation.isPending,
    loginError: loginMutation.error,
    setup: setupMutation.mutateAsync,
    isSettingUp: setupMutation.isPending,
    setupError: setupMutation.error,
    logout: logoutMutation.mutate,
    isLoggingOut: logoutMutation.isPending,
    refreshMe: meQuery.refetch,
  };
};
