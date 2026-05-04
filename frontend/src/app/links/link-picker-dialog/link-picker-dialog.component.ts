import { ChangeDetectionStrategy, Component, inject, resource, signal } from '@angular/core';
import { form, FormField, FormRoot, required } from '@angular/forms/signals';
import { MatButtonModule } from '@angular/material/button';
import { MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatIconModule } from '@angular/material/icon';
import { MatListModule } from '@angular/material/list';
import { MatRadioModule } from '@angular/material/radio';
import { firstValueFrom } from 'rxjs';
import type { Link } from '~/api/models/link';
import { LinksService } from '~/frontend/services/links.service';

@Component({
  selector: 'app-link-picker-dialog',
  templateUrl: './link-picker-dialog.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
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
  private readonly linksService = inject(LinksService);

  protected readonly listResource = resource({
    params: () => ({}),
    loader: () => firstValueFrom(this.linksService.listLinks()),
  });

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
          const links = this.listResource.value() ?? [];
          const link = links.find((l) => l.id === id);
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
