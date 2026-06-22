import { Clipboard, ClipboardModule } from '@angular/cdk/clipboard';
import { Component, inject, signal } from '@angular/core';
import { form, FormField, FormRoot } from '@angular/forms/signals';
import { MatButtonModule } from '@angular/material/button';
import { MatDatepickerModule } from '@angular/material/datepicker';
import { provideNativeDateAdapter } from '@angular/material/core';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { endOfDay, startOfTomorrow } from 'date-fns';
import { Api } from '~/api/api';
import { createPersonalAccessToken } from '~/api/functions';
import type { CreatePersonalAccessTokenRequest } from '~/api/models/create-personal-access-token-request';
import type { CreatePersonalAccessTokenResponse } from '~/api/models/create-personal-access-token-response';
import { snackErrorDuration, snackSuccessDuration } from '~/frontend/constants';

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
    ClipboardModule,
    FormField,
    FormRoot,
    MatButtonModule,
    MatDatepickerModule,
    MatDialogModule,
    MatFormFieldModule,
    MatIconModule,
    MatInputModule,
    MatSnackBarModule,
  ],
  templateUrl: './user-pat-create-dialog.component.html',
  styleUrl: './user-pat-create-dialog.component.css',
})
export class UserPatCreateDialogComponent {
  private readonly api = inject(Api);
  private readonly clipboard = inject(Clipboard);
  private readonly snackBar = inject(MatSnackBar);
  private readonly dialogRef = inject(
    MatDialogRef<UserPatCreateDialogComponent, CreatePersonalAccessTokenResponse | undefined>,
  );
  private readonly data = inject<UserPatCreateDialogData>(MAT_DIALOG_DATA, { optional: true });

  private response: CreatePersonalAccessTokenResponse | undefined;

  protected readonly errorMessage = signal<string | null>(null);
  protected readonly createdToken = signal<string | null>(null);
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
            // The picker yields local midnight; send end-of-day so the instant is
            // unambiguously in the future regardless of the client's UTC offset.
            expiresAt: expiresAt ? endOfDay(expiresAt).toISOString() : undefined,
          };
          if (trimmedLabel !== '') {
            body.label = trimmedLabel;
          }
          this.response = await this.api.invoke(createPersonalAccessToken, {
            body,
          });
          // The token is shown only once, so force an explicit dismissal to
          // guarantee the parent reloads the list with the new token.
          this.dialogRef.disableClose = true;
          this.createdToken.set(this.response.token);
        } catch (e) {
          console.error(e);
          this.errorMessage.set('Failed to create access token');
        }
      },
    },
  });

  protected copyCreatedToken(): void {
    const token = this.createdToken();
    if (!token) return;

    if (this.clipboard.copy(token)) {
      this.snackBar.open('Token copied', 'OK', { duration: snackSuccessDuration });
      return;
    }

    this.snackBar.open('Failed to copy token', 'Dismiss', { duration: snackErrorDuration });
  }

  protected done(): void {
    this.dialogRef.close(this.response);
  }

  protected cancel(): void {
    this.dialogRef.close(undefined);
  }
}
