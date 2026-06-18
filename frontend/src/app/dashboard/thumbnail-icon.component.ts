import {
  Component,
  OnDestroy,
  computed,
  effect,
  inject,
  input,
  resource,
  signal,
} from '@angular/core';
import { MatIconModule } from '@angular/material/icon';
import { Api } from '~/api/api';
import { getFileThumbnail } from '~/api/functions';
import { File } from '~/api/models';

const THUMBNAIL_MIME_TYPES = new Set([
  'application/pdf',
  'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
  'application/vnd.openxmlformats-officedocument.presentationml.presentation',
  'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
]);

function canHaveThumbnail(file: File): boolean {
  if (file.isDirectory || !file.checksum) return false;
  const mime = file.mimeType?.toLowerCase().trim();
  return mime !== undefined && (mime.startsWith('image/') || THUMBNAIL_MIME_TYPES.has(mime));
}

@Component({
  selector: 'app-thumbnail-icon',
  host: {
    class: 'flex items-center justify-center size-10',
  },
  template: `
    @if (thumbnailUrl(); as url) {
      <img
        [src]="url"
        alt=""
        draggable="false"
        class="mat-corner-sm block size-full object-cover"
        height="512"
        width="512"
      />
    } @else {
      <mat-icon>{{ iconName() }}</mat-icon>
    }
  `,
  imports: [MatIconModule],
})
export class ThumbnailIconComponent implements OnDestroy {
  private readonly api = inject(Api);

  public readonly file = input.required<File>();

  protected readonly iconName = computed(() => {
    const d = this.file();
    if (d.isDirectory) return 'folder';
    const mime = d.mimeType ?? '';
    if (mime.startsWith('image/')) return 'image';
    if (mime === 'application/pdf') return 'picture_as_pdf';
    return 'insert_drive_file';
  });

  private readonly thumbnailBlob = resource({
    params: () => ({ id: this.file().id, supported: canHaveThumbnail(this.file()) }),
    loader: async ({ params }) => {
      if (!params.supported) return null;
      try {
        return (await this.api.invoke(getFileThumbnail, { id: params.id })) as Blob;
      } catch {
        return null;
      }
    },
  });

  private objectUrl: string | null = null;
  protected readonly thumbnailUrl = signal<string | null>(null);

  constructor() {
    effect(() => {
      // Use effect() rather than computed(), because createObjectURL is a
      // side effect that must be revoked on each change and on destroy to
      // avoid memory leaks.
      const blob = this.thumbnailBlob.value();
      if (this.objectUrl) {
        URL.revokeObjectURL(this.objectUrl);
        this.objectUrl = null;
      }
      if (blob) {
        this.objectUrl = URL.createObjectURL(blob);
      }
      this.thumbnailUrl.set(this.objectUrl);
    });
  }

  public ngOnDestroy(): void {
    if (this.objectUrl) {
      URL.revokeObjectURL(this.objectUrl);
    }
  }
}
