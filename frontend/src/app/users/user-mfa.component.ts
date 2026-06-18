import { Clipboard } from '@angular/cdk/clipboard';
import { Component, inject, input, output, signal } from '@angular/core';
import { form, FormField, FormRoot, minLength, required } from '@angular/forms/signals';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { firstValueFrom } from 'rxjs';
import { snackErrorDuration, snackSuccessDuration } from '~/frontend/constants';
import type { MfaSetupData, SelfUserPort } from './user-data.port';

@Component({
  selector: 'app-user-mfa',
  imports: [
    FormField,
    FormRoot,
    MatButtonModule,
    MatCardModule,
    MatFormFieldModule,
    MatIconModule,
    MatInputModule,
    MatSnackBarModule,
  ],
  templateUrl: './user-mfa.component.html',
})
export class UserMfaComponent {
  private readonly snackBar = inject(MatSnackBar);
  private readonly clipboard = inject(Clipboard);

  readonly port = input.required<SelfUserPort>();
  readonly mfaEnabled = input.required<boolean>();
  readonly changed = output<void>();

  readonly setupData = signal<MfaSetupData | undefined>(undefined);
  readonly recoveryCodes = signal<string[] | undefined>(undefined);
  readonly loading = signal(false);

  readonly enableModel = signal({ code: '' });
  readonly enableForm = form(
    this.enableModel,
    (p) => {
      required(p.code);
      minLength(p.code, 6);
    },
    {
      submission: {
        action: async () => {
          try {
            const data = await firstValueFrom(this.port().enableMfa(this.enableModel().code));
            this.setupData.set(undefined);
            this.recoveryCodes.set(data.recoveryCodes);
            this.enableModel.set({ code: '' });
            this.enableForm().reset();
            this.snackBar.open('MFA enabled', 'OK', { duration: snackSuccessDuration });
            this.changed.emit();
          } catch {
            this.snackBar.open('Invalid verification code', 'OK', { duration: snackErrorDuration });
          }
        },
      },
    },
  );

  readonly disableModel = signal({ password: '' });
  readonly disableForm = form(
    this.disableModel,
    (p) => {
      required(p.password);
    },
    {
      submission: {
        action: async () => {
          try {
            await firstValueFrom(this.port().disableMfa(this.disableModel().password));
            this.disableModel.set({ password: '' });
            this.disableForm().reset();
            this.snackBar.open('MFA disabled', 'OK', { duration: snackSuccessDuration });
            this.changed.emit();
          } catch {
            this.snackBar.open('Invalid password', 'OK', { duration: snackErrorDuration });
          }
        },
      },
    },
  );

  setupMFA() {
    this.loading.set(true);
    this.port()
      .startMfaSetup()
      .subscribe({
        next: (data) => {
          this.loading.set(false);
          this.setupData.set(data);
        },
        error: () => {
          this.loading.set(false);
          this.snackBar.open('Failed to set up MFA', 'OK', { duration: snackErrorDuration });
        },
      });
  }

  copySecret() {
    const secret = this.setupData()?.secret;
    if (secret) {
      this.clipboard.copy(secret);
      this.snackBar.open('Secret copied', 'OK', { duration: snackSuccessDuration });
    }
  }

  copyRecoveryCodes() {
    const codes = this.recoveryCodes();
    if (codes) {
      this.clipboard.copy(codes.join('\n'));
      this.snackBar.open('Recovery codes copied', 'OK', { duration: snackSuccessDuration });
    }
  }
}
