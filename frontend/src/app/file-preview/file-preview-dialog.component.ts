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
import { DomSanitizer, SafeUrl } from '@angular/platform-browser';
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

  protected readonly loading = signal(true);
  protected readonly src = signal<SafeUrl | null>(null);

  private blob: Blob | null = null;
  private objectUrl: string | null = null;
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
      this.objectUrl = URL.createObjectURL(this.blob);
      this.src.set(this.sanitizer.bypassSecurityTrustUrl(this.objectUrl));
      this.loading.set(false);
    } catch (e) {
      if (token !== this.loadToken) return;
      console.error(e);
      this.snackBar.open('Failed to load image', 'Dismiss', { duration: snackErrorDuration });
      this.dialogRef.close();
    }
  }

  private revoke(): void {
    if (this.objectUrl) {
      URL.revokeObjectURL(this.objectUrl);
      this.objectUrl = null;
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
