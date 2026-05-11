export function buildShareUrl(accessToken: string): string {
  return `${window.location.origin}/share/${accessToken}`;
}
