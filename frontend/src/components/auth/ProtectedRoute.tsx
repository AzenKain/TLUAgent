import React from 'react';
import { Navigate, Outlet } from 'react-router-dom';
import { useAuth } from '@/hooks/useAuth';
import { LoadingSpinner } from '@/components/common/LoadingSpinner';
import {
  isAdminUser,
  isBannedUser,
  hasPermission,
  hasAnyPermission,
} from '@/utils/permission';

interface ProtectedRouteProps {
  requireAdmin?: boolean;
  requiredPermission?: string;
  requiredAnyPermissions?: string[];
  requiredRoles?: string[];
  redirectPath?: string;
}

export const ProtectedRoute: React.FC<ProtectedRouteProps> = ({
  requireAdmin = false,
  requiredPermission,
  requiredAnyPermissions,
  requiredRoles,
  redirectPath = '/login',
}) => {
  const { isAuthenticated, isSetupLoading, setupStatus, user } = useAuth();

  if (isSetupLoading) {
    return (
      <div className="h-screen flex items-center justify-center">
        <LoadingSpinner size="lg" />
      </div>
    );
  }

  if (setupStatus?.required) {
    return <Navigate to="/setup" replace />;
  }

  if (!isAuthenticated || !user || isBannedUser(user)) {
    return <Navigate to={redirectPath} replace />;
  }

  if (requireAdmin && !isAdminUser(user)) {
    return <Navigate to="/" replace />;
  }

  if (requiredRoles && requiredRoles.length > 0) {
    const normalizedRoles = requiredRoles.map((r) => r.toUpperCase());
    const hasRole =
      isAdminUser(user) ||
      user.roles?.some(
        (r) =>
          typeof r?.name === 'string' &&
          normalizedRoles.includes(r.name.toUpperCase())
      );
    if (!hasRole) {
      return <Navigate to="/" replace />;
    }
  }

  if (requiredPermission && !hasPermission(user, requiredPermission)) {
    return <Navigate to="/" replace />;
  }

  if (
    requiredAnyPermissions &&
    requiredAnyPermissions.length > 0 &&
    !hasAnyPermission(user, requiredAnyPermissions)
  ) {
    return <Navigate to="/" replace />;
  }

  return <Outlet />;
};
