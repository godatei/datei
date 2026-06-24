import { Component, inject, OnInit, signal } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatSnackBar } from '@angular/material/snack-bar';
import { Api } from '~/api/api';
import { snackErrorDuration } from '~/frontend/constants';
import { getFilePath, restoreTrash } from '~/api/functions';
import { File, FilePathItem } from '~/api/models';
import {
  DirectoryPickerComponent,
  DirectorySelection,
} from '~/frontend/components/directory-picker.component';

@Component({
  templateUrl: './restore-dialog.component.html',
  imports: [MatButtonModule, MatDialogModule, DirectoryPickerComponent],
})
export class RestoreDialogComponent implements OnInit {
  protected readonly data = inject<File>(MAT_DIALOG_DATA);
  protected readonly dialogRef = inject(MatDialogRef<RestoreDialogComponent>);
  private readonly api = inject(Api);
  private readonly snack = inject(MatSnackBar);

  protected readonly initialPath = signal<FilePathItem[]>([]);
  protected readonly selectedParent = signal<DirectorySelection | undefined>(undefined);

  public async ngOnInit(): Promise<void> {
    if (this.data.parentId) {
      try {
        // initialize the directory picker with the original parent unless it is also trashed
        const path = await this.api.invoke(getFilePath, { id: this.data.parentId });
        if (!path.some((it) => it.trashed)) {
          this.initialPath.set(path);
        }
      } catch (e) {
        console.error(e);
        this.snack.open('Failed to load original path', 'Dismiss', {
          duration: snackErrorDuration,
        });
      }
    }
  }

  protected async restore() {
    const parent = this.selectedParent();
    try {
      await this.api.invoke(restoreTrash, {
        fileId: this.data.id,
        body: { parentId: parent?.id ?? null },
      });
      this.dialogRef.close({ parent });
    } catch (e) {
      console.error(e);
      this.snack.open(
        `Failed to restore ${this.data.name ?? 'Unnamed'} in ${parent?.name ?? 'My files'}`,
        'Dismiss',
        {
          duration: snackErrorDuration,
        },
      );
    }
  }
}
