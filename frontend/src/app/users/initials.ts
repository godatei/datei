export function initials(name: string | null | undefined): string {
  if (!name) return '?';
  const letters = name
    .split(/\s+/)
    .filter(Boolean)
    .map((w) => w[0].toUpperCase())
    .slice(0, 3)
    .join('');
  return letters || '?';
}
