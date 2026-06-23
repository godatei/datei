import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  inject,
  signal,
} from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSnackBar } from '@angular/material/snack-bar';
import { DomSanitizer, SafeResourceUrl, SafeUrl } from '@angular/platform-browser';
import { Api } from '~/api/api';
import { downloadFile } from '~/api/functions';
import { File } from '~/api/models';
import { snackErrorDuration } from '~/frontend/constants';
import { triggerDownload } from '~/util/download';
import { isPreviewable } from '~/util/previewable';

export interface FilePreviewDialogData {
  files: File[];
  currentId: string;
}

@Component({
  selector: 'app-file-preview-dialog',
  templateUrl: './file-preview-dialog.component.html',
  styleUrl: './file-preview-dialog.component.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: {
    class: 'lightbox',
    '(click)': 'onBackdropClick($event)',
    '(keydown.arrowleft)': 'prev()',
    '(keydown.arrowright)': 'next()',
  },
  imports: [MatButtonModule, MatIconModule, MatProgressSpinnerModule],
})
export class FilePreviewDialogComponent {
  protected readonly data = inject<FilePreviewDialogData>(MAT_DIALOG_DATA);
  private readonly dialogRef = inject(MatDialogRef<FilePreviewDialogComponent>);
  private readonly api = inject(Api);
  private readonly sanitizer = inject(DomSanitizer);
  private readonly snackBar = inject(MatSnackBar);

  protected readonly items = this.data.files.filter(isPreviewable);
  protected readonly index = signal(
    Math.max(
      0,
      this.items.findIndex((f) => f.id === this.data.currentId),
    ),
  );
  protected readonly current = computed(() => this.items[this.index()]);
  protected readonly hasPrev = computed(() => this.index() > 0);
  protected readonly hasNext = computed(() => this.index() < this.items.length - 1);
  protected readonly multiple = this.items.length > 1;

  protected readonly kind = computed<'image' | 'pdf'>(() =>
    this.current().mimeType === 'application/pdf' ? 'pdf' : 'image',
  );

  protected readonly loading = signal(true);
  private readonly url = signal<string | null>(null);

  // The element rendering the blob dictates the sanitizer context: <img> is a
  // URL context, while <embed>/<iframe> are resource-URL contexts. Only the one
  // matching the current kind is non-null.
  protected readonly imageSrc = computed<SafeUrl | null>(() => {
    const u = this.url();
    return u !== null && this.kind() === 'image' ? this.sanitizer.bypassSecurityTrustUrl(u) : null;
  });
  protected readonly pdfSrc = computed<SafeResourceUrl | null>(() => {
    const u = this.url();
    return u !== null && this.kind() === 'pdf'
      ? this.sanitizer.bypassSecurityTrustResourceUrl(u)
      : null;
  });

  private blob: Blob | null = null;
  // Guards against an earlier, slower request overwriting the result of a more
  // recent navigation; only the latest token is allowed to commit.
  private loadToken = 0;

  constructor() {
    void this.load();
    inject(DestroyRef).onDestroy(() => this.revoke());
  }

  protected prev(): void {
    if (!this.hasPrev()) return;
    this.index.update((i) => i - 1);
    void this.load();
  }

  protected next(): void {
    if (!this.hasNext()) return;
    this.index.update((i) => i + 1);
    void this.load();
  }

  private async load(): Promise<void> {
    const item = this.current();
    const token = ++this.loadToken;
    this.loading.set(true);
    try {
      const response = await this.api.invoke$Response(downloadFile, { id: item.id });
      if (token !== this.loadToken) return;
      this.revoke();
      this.blob = response.body as Blob;
      this.url.set(URL.createObjectURL(this.blob));
      this.loading.set(false);
    } catch (e) {
      if (token !== this.loadToken) return;
      console.error(e);
      this.snackBar.open('Failed to load file', 'Dismiss', { duration: snackErrorDuration });
      this.dialogRef.close();
    }
  }

  private revoke(): void {
    const u = this.url();
    if (u !== null) {
      URL.revokeObjectURL(u);
      this.url.set(null);
    }
  }

  protected close(): void {
    this.dialogRef.close();
  }

  // Close only when the click lands on the backdrop itself, not on the image
  // or the toolbar controls nested inside it.
  protected onBackdropClick(event: MouseEvent): void {
    if (event.target === event.currentTarget) this.close();
  }

  protected download(): void {
    if (this.blob) triggerDownload(this.blob, this.current().name ?? 'download');
  }
}
