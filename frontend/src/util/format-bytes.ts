const UNITS = ['B', 'KB', 'MB', 'GB', 'TB'] as const;

/**
 * Formats a byte count as a human-readable string using 1024-base units.
 * Returns '' when bytes is null/undefined.
 */
export function formatBytes(bytes: number | null | undefined): string {
  if (bytes == null) return '';
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < UNITS.length - 1) {
    value /= 1024;
    unit++;
  }
  return unit === 0 ? `${value} ${UNITS[unit]}` : `${value.toFixed(1)} ${UNITS[unit]}`;
}
