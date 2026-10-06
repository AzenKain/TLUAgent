import { useAuthStore } from '@/stores/authStore';
import {
  isAdminUser,
  isBannedUser,
  hasPermission,
  hasAnyPermission,
  hasAllPermissions,
} from '@/utils/permission';

export const usePermissions = () => {
  const user = useAuthStore((state) => state.user);

  const isAdmin = isAdminUser(user);
  const isBanned = isBannedUser(user);

  const hasRole = (roleName: string) => {
    return Boolean(user?.roles?.some((r) => r.name.toLowerCase() === roleName.toLowerCase()));
  };

  const can = (permissionKey: string) => hasPermission(user, permissionKey);
  const canAny = (permissionKeys: string[]) => hasAnyPermission(user, permissionKeys);
  const canAll = (permissionKeys: string[]) => hasAllPermissions(user, permissionKeys);

  return {
    user,
    isAdmin,
    isBanned,
    hasRole,
    can,
    canAny,
    canAll,
    hasPermission: can,
    hasAnyPermission: canAny,
    hasAllPermissions: canAll,
  };
};
