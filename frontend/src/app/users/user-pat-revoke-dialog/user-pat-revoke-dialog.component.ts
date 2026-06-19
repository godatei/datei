import { Component, inject } from '@angular/core';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';

interface UserPatRevokeDialogData {
  label: string;
}

@Component({
  selector: 'app-user-pat-revoke-dialog',
  imports: [MatButtonModule, MatDialogModule],
  templateUrl: './user-pat-revoke-dialog.component.html',
})
export class UserPatRevokeDialogComponent {
  protected readonly data = inject<UserPatRevokeDialogData>(MAT_DIALOG_DATA);
  private readonly dialogRef = inject(MatDialogRef<UserPatRevokeDialogComponent, boolean>);

  protected cancel(): void {
    this.dialogRef.close(false);
  }

  protected confirm(): void {
    this.dialogRef.close(true);
  }
}
