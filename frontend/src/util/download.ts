/**
 * Triggers a browser download for the given Blob with the given filename.
 *
 * Uses an off-DOM anchor + object URL; the URL is revoked immediately after
 * `click()` since `click()` synchronously kicks off the download stream.
 */
export function triggerDownload(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}
