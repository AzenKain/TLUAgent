import type { User } from '@/types/auth';

export function isAdminUser(user: User | null | undefined): boolean {
  if (!user || !Array.isArray(user.roles)) return false;
  if (user.is_owner) return true;
  return user.roles.some(
    (role) =>
      Boolean(role?.is_admin) ||
      (typeof role?.name === 'string' && role.name.toUpperCase() === 'ADMIN')
  );
}

export function isBannedUser(user: User | null | undefined): boolean {
  if (!user || !Array.isArray(user.roles)) return false;
  return user.roles.some(
    (role) =>
      Boolean(role?.is_banned) ||
      (typeof role?.name === 'string' && role.name.toUpperCase() === 'BANNED')
  );
}

export function hasPermission(
  user: User | null | undefined,
  permissionKey: string
): boolean {
  if (typeof permissionKey !== 'string' || !permissionKey) return false;
  if (!user || !Array.isArray(user.roles) || user.roles.length === 0) return false;
  if (isBannedUser(user)) return false;
  if (isAdminUser(user)) return true;

  const sortedRoles = [...user.roles].filter(Boolean).toSorted((a, b) => {
    const posA = a.position ?? 0;
    const posB = b.position ?? 0;
    return posB - posA;
  });

  let allowed = false;
  for (const role of sortedRoles) {
    if (Array.isArray(role.permissions)) {
      for (const p of role.permissions) {
        if (!p || p.permission_key !== permissionKey) continue;
        if (p.effect === 'deny') {
          return false;
        }
        if (p.effect === 'allow') {
          allowed = true;
        }
      }
    }
  }

  return allowed;
}

export function hasAnyPermission(
  user: User | null | undefined,
  permissionKeys: string[]
): boolean {
  if (!permissionKeys || permissionKeys.length === 0) return true;
  return permissionKeys.some((key) => hasPermission(user, key));
}

export function hasAllPermissions(
  user: User | null | undefined,
  permissionKeys: string[]
): boolean {
  if (!permissionKeys || permissionKeys.length === 0) return true;
  return permissionKeys.every((key) => hasPermission(user, key));
}
