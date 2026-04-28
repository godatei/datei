import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { SafeUrl } from '@angular/platform-browser';

export interface ImagePreviewDialogData {
  src: SafeUrl;
  name: string;
}

@Component({
  selector: 'app-image-preview-dialog',
  templateUrl: './image-preview-dialog.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [MatDialogModule, MatButtonModule],
})
export class ImagePreviewDialogComponent {
  protected readonly data = inject<ImagePreviewDialogData>(MAT_DIALOG_DATA);
  private readonly dialogRef = inject(MatDialogRef<ImagePreviewDialogComponent>);

  protected close(): void {
    this.dialogRef.close();
  }
}
