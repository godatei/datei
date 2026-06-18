import { Component, computed, inject, signal } from '@angular/core';
import { form, FormField, FormRoot, maxLength, pattern, required } from '@angular/forms/signals';
import { MatButtonModule } from '@angular/material/button';
import { MatDatepickerModule } from '@angular/material/datepicker';
import { provideNativeDateAdapter } from '@angular/material/core';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatListModule } from '@angular/material/list';
import { startOfTomorrow } from 'date-fns';
import { Api } from '~/api/api';
import { createLink, removeFileFromLink, updateLink } from '~/api/functions';
import type { File } from '~/api/models/file';
import type { LinkDetail } from '~/api/models/link-detail';
import type { UpdateLinkRequest } from '~/api/models/update-link-request';

export type LinkFormDialogData =
  | { mode: 'create'; fileIds: string[]; defaultName?: string }
  | { mode: 'edit'; link: LinkDetail };

interface LinkFormModel {
  name: string;
  expiresAt: Date | null;
  code: string;
}

@Component({
  selector: 'app-link-form-dialog',
  templateUrl: './link-form-dialog.component.html',
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

  protected readonly isEdit = this.data.mode === 'edit';
  protected readonly title = this.isEdit ? 'Edit public link' : 'Create public link';
  protected readonly submitLabel = this.isEdit ? 'Save' : 'Create link';

  // Capture mode-specific payload up front so the rest of the class can use the
  // simpler `this.isEdit` boolean without re-narrowing `this.data` everywhere.
  private readonly editLink: LinkDetail | null = this.data.mode === 'edit' ? this.data.link : null;
  private readonly createFileIds: string[] = this.data.mode === 'create' ? this.data.fileIds : [];
  private readonly defaultName: string | undefined =
    this.data.mode === 'create' ? this.data.defaultName : undefined;

  protected readonly fileCount = computed(() => this.createFileIds.length);

  protected readonly errorMessage = signal<string | null>(null);
  protected readonly sharedFiles = signal<File[]>(this.editLink ? [...this.editLink.files] : []);
  // File IDs the user removed in this dialog session. Pending until Save —
  // Cancel/Escape/backdrop must leave the server state unchanged.
  private readonly pendingRemovals = signal<ReadonlySet<string>>(new Set());

  protected readonly model = signal<LinkFormModel>(this.initialModel());

  protected readonly tomorrow = startOfTomorrow();

  protected readonly linkForm = form(
    this.model,
    (p) => {
      required(p.name);
      maxLength(p.name, 255);
      pattern(p.name, /\S/);
      pattern(p.code, /\S/);
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
        code: v.code.trim() === '' ? undefined : v.code.trim(),
        fileIds: this.createFileIds,
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
      body.code = v.code.trim();
    } else {
      body.clearCode = true;
    }

    // submitEdit is only invoked when isEdit, which is equivalent to editLink !== null.
    const linkId = this.editLink!.id;

    // Flush the queued removals first so the link's content set matches what
    // the dialog has been showing the user before the metadata update lands.
    for (const fileId of this.pendingRemovals()) {
      await this.api.invoke(removeFileFromLink, { id: linkId, fileId });
    }

    return this.api.invoke(updateLink, { id: linkId, body });
  }

  protected removeFile(file: File): void {
    if (!this.isEdit) return;
    this.sharedFiles.update((items) => items.filter((d) => d.id !== file.id));
    this.pendingRemovals.update((s) => new Set(s).add(file.id));
  }

  protected cancel(): void {
    this.dialogRef.close(undefined);
  }
}
