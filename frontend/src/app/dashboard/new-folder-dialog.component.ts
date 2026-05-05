import {
  ChangeDetectionStrategy,
  Component,
  ElementRef,
  inject,
  signal,
  viewChild,
} from '@angular/core';
import { form, FormField, FormRoot, maxLength, pattern, required } from '@angular/forms/signals';
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
    FormField,
    FormRoot,
  ],
})
export class NewFolderDialogComponent {
  private readonly dialogRef = inject(MatDialogRef<NewFolderDialogComponent, string | null>);
  private readonly nameInput = viewChild.required<ElementRef<HTMLInputElement>>('nameInput');

  protected readonly model = signal({ name: 'Untitled folder' });
  protected readonly form = form(
    this.model,
    (p) => {
      required(p.name);
      maxLength(p.name, 255);
      pattern(p.name, /\S/);
    },
    {
      submission: {
        action: async () => {
          this.dialogRef.close(this.model().name.trim());
        },
      },
    },
  );

  constructor() {
    this.dialogRef.afterOpened().subscribe(() => {
      const input = this.nameInput().nativeElement;
      input.focus();
      input.select();
    });
  }

  protected cancel(): void {
    this.dialogRef.close(null);
  }
}
