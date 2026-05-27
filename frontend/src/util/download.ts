// Triggers a browser download for the given Blob with the given filename via
// an off-DOM anchor + object URL. The revoke is deferred to the next tick so
// the browser has a chance to start the download stream before the URL goes
// away (synchronous revoke is unreliable across engines).
export function triggerDownload(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.click();
  setTimeout(() => URL.revokeObjectURL(url), 0);
}
