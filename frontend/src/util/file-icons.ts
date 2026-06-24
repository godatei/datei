import { MatIconRegistry } from '@angular/material/icon';
import { DomSanitizer } from '@angular/platform-browser';
import { File } from '~/api/models';
import { LSICON_SVGS } from './lsicon-svgs.generated';

export const LSICON_NAMESPACE = 'lsicon';

/** Material Symbols ligature used when a file's type maps to no lsicon icon. */
export const FILE_ICON_FALLBACK = 'draft';

/** Material Symbols ligature used for directories. */
export const FOLDER_ICON = 'folder';

type FileCategory =
  | 'folder'
  | 'image'
  | 'audio'
  | 'video'
  | 'pdf'
  | 'csv'
  | 'word'
  | 'excel'
  | 'ppt'
  | 'zip'
  | 'rar'
  | 'text'
  | 'other';

/** lsicon name per category (`null` falls back to a Material Symbols glyph). */
const CATEGORY_ICON: Record<FileCategory, string | null> = {
  folder: null,
  image: 'picture-filled',
  audio: 'music-filled',
  video: 'file-mp4-filled',
  pdf: 'file-pdf-filled',
  csv: 'file-csv-filled',
  word: 'file-doc-filled',
  excel: 'file-xls-filled',
  ppt: 'file-ppt-filled',
  zip: 'file-zip-filled',
  rar: 'file-rar-filled',
  text: 'file-txt-filled',
  other: null,
};

/** Accent color per category — no Material token exists for file-type accents. */
const CATEGORY_COLOR: Record<FileCategory, string> = {
  folder: '#9aa0a6', // muted, a touch lighter than the row text
  image: '#ec4899', // pink
  audio: '#9333ea', // purple
  video: '#6366f1', // indigo
  pdf: '#ef4444', // red
  csv: '#16a34a', // green
  word: '#2563eb', // blue
  excel: '#16a34a', // green
  ppt: '#ea580c', // orange
  zip: '#d97706', // amber
  rar: '#d97706', // amber
  text: '#64748b', // slate
  other: '#64748b', // slate
};

function categorize(file: File): FileCategory {
  if (file.isDirectory) return 'folder';
  const mime = file.mimeType?.toLowerCase().trim();
  if (!mime) return 'other';
  if (mime.startsWith('image/')) return 'image';
  if (mime.startsWith('audio/')) return 'audio';
  if (mime.startsWith('video/')) return 'video';
  if (mime.includes('pdf')) return 'pdf';
  if (mime.includes('csv')) return 'csv';
  if (mime.includes('word')) return 'word';
  if (mime.includes('sheet') || mime.includes('excel')) return 'excel';
  if (mime.includes('presentation') || mime.includes('powerpoint')) return 'ppt';
  if (mime.includes('zip')) return 'zip';
  if (mime.includes('rar')) return 'rar';
  if (mime.startsWith('text/')) return 'text';
  return 'other';
}

/** lsicon name for a file, or `null` to fall back to a Material Symbols icon. */
export function lsiconForFile(file: File): string | null {
  return CATEGORY_ICON[categorize(file)];
}

/** Accent color for a file's icon. */
export function fileIconColor(file: File): string {
  return CATEGORY_COLOR[categorize(file)];
}

/** Registers the pre-rendered file-type lsicon SVGs with Material's registry. */
export function registerFileLsicons(registry: MatIconRegistry, sanitizer: DomSanitizer): void {
  for (const [name, svg] of Object.entries(LSICON_SVGS)) {
    registry.addSvgIconLiteralInNamespace(
      LSICON_NAMESPACE,
      name,
      sanitizer.bypassSecurityTrustHtml(svg),
    );
  }
}
