import { Component, inject, signal } from '@angular/core';
import { form, FormField, FormRoot } from '@angular/forms/signals';
import { MatButtonModule } from '@angular/material/button';
import { MatDatepickerModule } from '@angular/material/datepicker';
import { provideNativeDateAdapter } from '@angular/material/core';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { startOfTomorrow } from 'date-fns';
import { Api } from '~/api/api';
import { createPersonalAccessToken } from '~/api/functions';
import type { CreatePersonalAccessTokenRequest } from '~/api/models/create-personal-access-token-request';
import type { CreatePersonalAccessTokenResponse } from '~/api/models/create-personal-access-token-response';

interface UserPatCreateDialogData {
  initialLabel?: string;
}

interface UserPatCreateFormModel {
  label: string;
  expiresAt: Date | null;
}

@Component({
  selector: 'app-user-pat-create-dialog',
  providers: [provideNativeDateAdapter()],
  imports: [
    FormField,
    FormRoot,
    MatButtonModule,
    MatDatepickerModule,
    MatDialogModule,
    MatFormFieldModule,
    MatInputModule,
  ],
  templateUrl: './user-pat-create-dialog.component.html',
})
export class UserPatCreateDialogComponent {
  private readonly api = inject(Api);
  private readonly dialogRef = inject(
    MatDialogRef<UserPatCreateDialogComponent, CreatePersonalAccessTokenResponse | undefined>,
  );
  private readonly data = inject<UserPatCreateDialogData>(MAT_DIALOG_DATA, { optional: true });

  protected readonly errorMessage = signal<string | null>(null);
  protected readonly tomorrow = startOfTomorrow();
  protected readonly model = signal<UserPatCreateFormModel>({
    label: this.data?.initialLabel ?? '',
    expiresAt: null,
  });

  protected readonly tokenForm = form(this.model, {
    submission: {
      action: async () => {
        this.errorMessage.set(null);
        try {
          const { label, expiresAt } = this.model();
          const trimmedLabel = label.trim();
          const body: CreatePersonalAccessTokenRequest = {
            expiresAt: expiresAt ? expiresAt.toISOString() : undefined,
          };
          if (trimmedLabel !== '') {
            body.label = trimmedLabel;
          }
          const response = await this.api.invoke(createPersonalAccessToken, {
            body,
          });
          this.dialogRef.close(response);
        } catch (e) {
          console.error(e);
          this.errorMessage.set('Failed to create access token');
        }
      },
    },
  });

  protected cancel(): void {
    this.dialogRef.close(undefined);
  }
}
