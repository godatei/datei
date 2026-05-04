import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { FormControl, ReactiveFormsModule, Validators } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';

export interface RenameDateiDialogData {
  currentName: string;
}

@Component({
  selector: 'app-rename-datei-dialog',
  templateUrl: './rename-datei-dialog.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    ReactiveFormsModule,
  ],
})
export class RenameDateiDialogComponent {
  private readonly dialogRef = inject(MatDialogRef<RenameDateiDialogComponent, string | null>);
  protected readonly data = inject<RenameDateiDialogData>(MAT_DIALOG_DATA);

  protected readonly nameControl = new FormControl(this.data.currentName, {
    nonNullable: true,
    validators: [Validators.required, Validators.maxLength(255), Validators.pattern(/\S/)],
  });

  protected confirm(): void {
    if (this.nameControl.invalid) return;
    const next = this.nameControl.value.trim();
    if (next === this.data.currentName) {
      this.dialogRef.close(null);
      return;
    }
    this.dialogRef.close(next);
  }

  protected cancel(): void {
    this.dialogRef.close(null);
  }
}
