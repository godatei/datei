export function buildShareUrl(key: string): string {
  return `${window.location.origin}/share/${key}`;
}
