import {
  ChangeDetectionStrategy,
  Component,
  computed,
  ElementRef,
  inject,
  signal,
  viewChild,
} from '@angular/core';
import { form, FormField, FormRoot, maxLength, pattern, required } from '@angular/forms/signals';
import { MatButtonModule } from '@angular/material/button';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';

export interface RenameDateiDialogData {
  currentName: string;
  isDirectory: boolean;
}

function extensionOf(name: string): string {
  const lastDot = name.lastIndexOf('.');
  return lastDot > 0 ? name.slice(lastDot) : '';
}

@Component({
  selector: 'app-rename-datei-dialog',
  templateUrl: './rename-datei-dialog.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatIconModule,
    MatInputModule,
    FormField,
    FormRoot,
  ],
})
export class RenameDateiDialogComponent {
  private readonly dialogRef = inject(MatDialogRef<RenameDateiDialogComponent, string | null>);
  protected readonly data = inject<RenameDateiDialogData>(MAT_DIALOG_DATA);
  private readonly nameInput = viewChild.required<ElementRef<HTMLInputElement>>('nameInput');

  protected readonly model = signal({ name: this.data.currentName });
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
          const next = this.model().name.trim();
          this.dialogRef.close(next === this.data.currentName ? null : next);
        },
      },
    },
  );

  protected readonly currentExt = computed(() =>
    this.data.isDirectory ? '' : extensionOf(this.data.currentName),
  );
  protected readonly nextExt = computed(() =>
    this.data.isDirectory ? '' : extensionOf(this.model().name),
  );
  protected readonly extensionChanged = computed(() => this.currentExt() !== this.nextExt());

  constructor() {
    this.dialogRef.afterOpened().subscribe(() => {
      const input = this.nameInput().nativeElement;
      input.focus();
      const lastDot = this.data.isDirectory ? -1 : this.data.currentName.lastIndexOf('.');
      if (lastDot > 0) {
        input.setSelectionRange(0, lastDot);
      } else {
        input.select();
      }
    });
  }

  protected cancel(): void {
    this.dialogRef.close(null);
  }
}
