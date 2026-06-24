import { inject, Injectable } from '@angular/core';
import { MatDialog, MatDialogRef } from '@angular/material/dialog';
import { File } from '~/api/models';
import { FilePreviewDialogComponent, FilePreviewDialogData } from './file-preview-dialog.component';

@Injectable({ providedIn: 'root' })
export class FilePreviewService {
  private readonly dialog = inject(MatDialog);

  open(files: File[], currentId: string): MatDialogRef<FilePreviewDialogComponent> {
    return this.dialog.open(FilePreviewDialogComponent, {
      data: { files, currentId } satisfies FilePreviewDialogData,
      panelClass: 'lightbox-panel',
      backdropClass: 'lightbox-backdrop',
      width: '100vw',
      height: '100vh',
      maxWidth: '100vw',
    });
  }
}
