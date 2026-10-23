
export const Permission: Record<number, string> = {
  1: 'Create new image',
  [1 << 1]: 'Edit image',
  [1 << 2]: 'Issue deletion',
  [1 << 3]: 'Admin',
}

export const ADMIN_PERMISSION = 1 << 3;

export function hasAdminPermission(permission: number | undefined) {
  return permission !== undefined && (permission & ADMIN_PERMISSION) === ADMIN_PERMISSION;
}

export function formatPermissions(permission: number) {
  return [
    permission & ADMIN_PERMISSION ? 'A' : '-',
    permission & (1 << 2) ? 'D' : '-',
    permission & (1 << 1) ? 'E' : '-',
    permission & 1 ? 'C' : '-',
  ].join('');
}
