import { File } from '~/api/models';

// Whether a file can be shown in the preview lightbox. Currently limited to
// images; extend here when other media types become previewable.
export function isPreviewable(file: File): boolean {
  return file.mimeType?.startsWith('image/') ?? false;
}
