import { ChangeDetectionStrategy, Component, ElementRef, inject, viewChild } from '@angular/core';
import { FormControl, ReactiveFormsModule, Validators } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';

@Component({
  selector: 'app-new-folder-dialog',
  templateUrl: './new-folder-dialog.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    ReactiveFormsModule,
  ],
})
export class NewFolderDialogComponent {
  private readonly dialogRef = inject(MatDialogRef<NewFolderDialogComponent>);
  private readonly nameInput = viewChild.required<ElementRef<HTMLInputElement>>('nameInput');

  protected readonly nameControl = new FormControl('Untitled folder', {
    nonNullable: true,
    validators: [Validators.required, Validators.maxLength(255), Validators.pattern(/\S/)],
  });

  constructor() {
    this.dialogRef.afterOpened().subscribe(() => {
      const input = this.nameInput().nativeElement;
      input.focus();
      input.select();
    });
  }

  protected confirm(): void {
    if (this.nameControl.invalid) return;
    this.dialogRef.close(this.nameControl.value.trim());
  }

  protected cancel(): void {
    this.dialogRef.close(null);
  }
}
