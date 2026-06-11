import { Component, computed, inject, resource, signal } from '@angular/core';
import { form, FormField, FormRoot, required } from '@angular/forms/signals';
import { MatButtonModule } from '@angular/material/button';
import { MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatIconModule } from '@angular/material/icon';
import { MatListModule } from '@angular/material/list';
import { MatRadioModule } from '@angular/material/radio';
import { Api } from '~/api/api';
import { listLinks } from '~/api/functions';
import type { Link } from '~/api/models/link';

@Component({
  selector: 'app-link-picker-dialog',
  templateUrl: './link-picker-dialog.component.html',
  imports: [
    MatDialogModule,
    MatButtonModule,
    MatIconModule,
    MatListModule,
    MatRadioModule,
    FormField,
    FormRoot,
  ],
})
export class LinkPickerDialogComponent {
  private readonly dialogRef = inject(MatDialogRef<LinkPickerDialogComponent, Link | undefined>);
  private readonly api = inject(Api);

  protected readonly listResource = resource({
    params: () => ({}),
    loader: () => this.api.invoke(listLinks, { status: 'active' }),
  });

  protected readonly availableLinks = computed(() => this.listResource.value()?.items ?? []);

  protected readonly model = signal({ linkId: '' });
  protected readonly pickerForm = form(
    this.model,
    (p) => {
      required(p.linkId);
    },
    {
      submission: {
        action: async () => {
          const id = this.model().linkId;
          const link = this.availableLinks().find((l) => l.id === id);
          if (link) {
            this.dialogRef.close(link);
          }
        },
      },
    },
  );

  protected cancel(): void {
    this.dialogRef.close(undefined);
  }
}
