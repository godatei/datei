import { HttpErrorResponse } from '@angular/common/http';
import { Component, computed, inject, signal } from '@angular/core';
import { form, FormField, FormRoot, required } from '@angular/forms/signals';
import { MatButtonModule } from '@angular/material/button';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { Api } from '~/api/api';
import { createMailAccount, updateMailAccount } from '~/api/functions';
import type { MailAccount } from '~/api/models/mail-account';
import { retryOnConflict } from '~/util/retry-on-conflict';

export interface MailAccountDialogData {
  account?: MailAccount;
}

type Security = 'ssl' | 'starttls' | 'none';

const CANONICAL_PORTS: Record<Security, number> = {
  ssl: 993,
  starttls: 143,
  none: 143,
};
const CANONICAL_PORT_VALUES = new Set(Object.values(CANONICAL_PORTS));

interface MailAccountFormModel {
  name: string;
  imapHost: string;
  imapPort: number;
  username: string;
  password: string;
  security: Security;
}

@Component({
  selector: 'app-mail-account-dialog',
  templateUrl: './mail-account-dialog.component.html',
  imports: [
    FormField,
    FormRoot,
    MatButtonModule,
    MatDialogModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
  ],
})
export class MailAccountDialogComponent {
  private readonly api = inject(Api);
  private readonly data = inject<MailAccountDialogData>(MAT_DIALOG_DATA, { optional: true });
  private readonly dialogRef = inject(
    MatDialogRef<MailAccountDialogComponent, MailAccount | undefined>,
  );

  protected readonly isEdit = computed(() => this.data?.account !== undefined);
  protected readonly errorMessage = signal<string | null>(null);

  protected readonly model = signal<MailAccountFormModel>({
    name: this.data?.account?.name ?? '',
    imapHost: this.data?.account?.imapHost ?? '',
    imapPort: this.data?.account?.imapPort ?? 993,
    username: this.data?.account?.username ?? '',
    password: '',
    security: (this.data?.account?.security as Security) ?? 'ssl',
  });

  protected readonly accountForm = form(
    this.model,
    (p) => {
      required(p.name);
      required(p.imapHost);
      required(p.username);
    },
    {
      submission: {
        action: async () => {
          this.errorMessage.set(null);
          const m = this.model();
          const password = m.password.trim();

          if (!this.isEdit() && password === '') {
            this.errorMessage.set('Password is required');
            return;
          }
          if (m.imapPort < 1 || m.imapPort > 65535) {
            this.errorMessage.set('Port must be between 1 and 65535');
            return;
          }

          try {
            const account = this.data?.account;
            if (account) {
              const result = await retryOnConflict(() =>
                this.api.invoke(updateMailAccount, {
                  accountId: account.id,
                  body: {
                    name: m.name.trim(),
                    imapHost: m.imapHost.trim(),
                    imapPort: m.imapPort,
                    username: m.username.trim(),
                    security: m.security,
                    ...(password !== '' ? { password } : {}),
                  },
                }),
              );
              this.dialogRef.close(result);
            } else {
              const result = await retryOnConflict(() =>
                this.api.invoke(createMailAccount, {
                  body: {
                    name: m.name.trim(),
                    imapHost: m.imapHost.trim(),
                    imapPort: m.imapPort,
                    username: m.username.trim(),
                    password,
                    security: m.security,
                  },
                }),
              );
              this.dialogRef.close(result);
            }
          } catch (e) {
            console.error(e);
            this.errorMessage.set(this.errorText(e));
          }
        },
      },
    },
  );

  private errorText(e: unknown): string {
    if (e instanceof HttpErrorResponse && typeof e.error?.message === 'string') {
      return e.error.message;
    }
    return 'Failed to save mail account';
  }

  // When switching security, move the port to the new option's canonical port,
  // but only if the user hasn't set a custom (non-canonical) port.
  protected onSecurityChange(security: Security): void {
    if (CANONICAL_PORT_VALUES.has(this.model().imapPort)) {
      this.model.update((m) => ({ ...m, imapPort: CANONICAL_PORTS[security] }));
    }
  }

  protected cancel(): void {
    this.dialogRef.close(undefined);
  }
}
