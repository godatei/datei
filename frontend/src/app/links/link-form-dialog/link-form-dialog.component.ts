import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { form, FormField, FormRoot, maxLength, pattern, required } from '@angular/forms/signals';
import { MatButtonModule } from '@angular/material/button';
import { MatDatepickerModule } from '@angular/material/datepicker';
import { provideNativeDateAdapter } from '@angular/material/core';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatListModule } from '@angular/material/list';
import { MatSnackBar } from '@angular/material/snack-bar';
import { startOfTomorrow } from 'date-fns';
import { Api } from '~/api/api';
import { createLink, removeDateiFromLink, updateLink } from '~/api/functions';
import type { Datei } from '~/api/models/datei';
import type { LinkDetail } from '~/api/models/link-detail';
import type { UpdateLinkRequest } from '~/api/models/update-link-request';

export type LinkFormDialogData =
  | { mode: 'create'; dateiIds: string[]; defaultName?: string }
  | { mode: 'edit'; link: LinkDetail };

interface LinkFormModel {
  name: string;
  expiresAt: Date | null;
  code: string;
}

@Component({
  selector: 'app-link-form-dialog',
  templateUrl: './link-form-dialog.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
  providers: [provideNativeDateAdapter()],
  imports: [
    MatDialogModule,
    MatButtonModule,
    MatFormFieldModule,
    MatIconModule,
    MatInputModule,
    MatListModule,
    MatDatepickerModule,
    FormField,
    FormRoot,
  ],
})
export class LinkFormDialogComponent {
  protected readonly data = inject<LinkFormDialogData>(MAT_DIALOG_DATA);
  private readonly dialogRef = inject(
    MatDialogRef<LinkFormDialogComponent, LinkDetail | undefined>,
  );
  private readonly api = inject(Api);
  private readonly snackBar = inject(MatSnackBar);

  protected readonly isEdit = this.data.mode === 'edit';
  protected readonly title = this.isEdit ? 'Edit public link' : 'Create public link';
  protected readonly submitLabel = this.isEdit ? 'Save' : 'Create link';

  // Capture mode-specific payload up front so the rest of the class can use the
  // simpler `this.isEdit` boolean without re-narrowing `this.data` everywhere.
  private readonly editLink: LinkDetail | null = this.data.mode === 'edit' ? this.data.link : null;
  private readonly createDateiIds: string[] = this.data.mode === 'create' ? this.data.dateiIds : [];
  private readonly defaultName: string | undefined =
    this.data.mode === 'create' ? this.data.defaultName : undefined;

  protected readonly dateiCount = computed(() => this.createDateiIds.length);

  protected readonly errorMessage = signal<string | null>(null);
  protected readonly sharedDateien = signal<Datei[]>(
    this.editLink ? [...this.editLink.dateien] : [],
  );
  protected readonly removingDateiId = signal<string | null>(null);
  // Tracks whether any datei was removed; used so the parent refreshes even if
  // the user dismisses the dialog without saving the form.
  private modified = false;

  protected readonly model = signal<LinkFormModel>(this.initialModel());

  protected readonly tomorrow = startOfTomorrow();

  protected readonly linkForm = form(
    this.model,
    (p) => {
      required(p.name);
      maxLength(p.name, 255);
      pattern(p.name, /\S/);
    },
    {
      submission: {
        action: async () => {
          this.errorMessage.set(null);
          try {
            const result = this.isEdit ? await this.submitEdit() : await this.submitCreate();
            this.dialogRef.close(result);
          } catch (e) {
            console.error(e);
            this.errorMessage.set(this.isEdit ? 'Failed to update link' : 'Failed to create link');
          }
        },
      },
    },
  );

  private initialModel(): LinkFormModel {
    if (this.editLink) {
      const l = this.editLink;
      return {
        name: l.name,
        expiresAt: l.expiresAt ? new Date(l.expiresAt) : null,
        code: l.code ?? '',
      };
    }
    return {
      name: this.defaultName ?? 'Shared files',
      expiresAt: null,
      code: '',
    };
  }

  private async submitCreate(): Promise<LinkDetail> {
    const v = this.model();
    return this.api.invoke(createLink, {
      body: {
        name: v.name.trim(),
        expiresAt: v.expiresAt ? v.expiresAt.toISOString() : undefined,
        code: v.code.trim() === '' ? undefined : v.code,
        dateiIds: this.createDateiIds,
      },
    });
  }

  private async submitEdit(): Promise<LinkDetail> {
    const v = this.model();
    const body: UpdateLinkRequest = { name: v.name.trim() };

    if (v.expiresAt) {
      body.expiresAt = v.expiresAt.toISOString();
    } else {
      body.clearExpiration = true;
    }

    if (v.code.trim() !== '') {
      body.code = v.code;
    } else {
      body.clearCode = true;
    }

    // submitEdit is only invoked when isEdit, which is equivalent to editLink !== null.
    return this.api.invoke(updateLink, { id: this.editLink!.id, body });
  }

  protected async removeDatei(datei: Datei): Promise<void> {
    if (!this.isEdit) return;
    if (this.removingDateiId() !== null) return;

    this.removingDateiId.set(datei.id);
    try {
      await this.api.invoke(removeDateiFromLink, {
        id: this.editLink!.id,
        dateiId: datei.id,
      });
      this.sharedDateien.update((items) => items.filter((d) => d.id !== datei.id));
      this.modified = true;
    } catch (e) {
      console.error(e);
      this.snackBar.open('Failed to remove item from link', 'Dismiss', { duration: 4000 });
    } finally {
      this.removingDateiId.set(null);
    }
  }

  protected cancel(): void {
    // If a datei was removed, signal the parent to refresh its list by closing
    // with the original link reference (truthy). Otherwise close with undefined
    // so the parent does not refresh unnecessarily.
    if (this.modified && this.isEdit) {
      this.dialogRef.close(this.editLink!);
    } else {
      this.dialogRef.close(undefined);
    }
  }
}
