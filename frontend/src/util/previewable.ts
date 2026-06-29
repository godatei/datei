import { File } from '~/api/models';

// Whether a file can be shown in the preview lightbox. Extend here when other
// media types become previewable.
export function isPreviewable(file: File): boolean {
  const mime = file.mimeType;
  return mime != null && (mime.startsWith('image/') || mime === 'application/pdf');
}
