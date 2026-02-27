
export const Permission: Record<number, string> = {
  1: 'Create new image',
  [1 << 1]: 'Edit image',
  [1 << 2]: 'Issue deletion',
}