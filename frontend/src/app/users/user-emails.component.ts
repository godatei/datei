import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  input,
  resource,
  signal,
} from '@angular/core';
import { email, form, FormField, FormRoot, required } from '@angular/forms/signals';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatChipsModule } from '@angular/material/chips';
import { MatDividerModule } from '@angular/material/divider';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { firstValueFrom } from 'rxjs';
import { snackErrorDuration, snackSuccessDuration } from '~/frontend/constants';
import type { BaseUserPort } from './user-data.port';

@Component({
  selector: 'app-user-emails',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    FormField,
    FormRoot,
    MatButtonModule,
    MatCardModule,
    MatChipsModule,
    MatDividerModule,
    MatFormFieldModule,
    MatIconModule,
    MatInputModule,
    MatSnackBarModule,
  ],
  templateUrl: './user-emails.component.html',
})
export class UserEmailsComponent {
  private readonly snackBar = inject(MatSnackBar);

  readonly port = input.required<BaseUserPort>();

  protected readonly emailsResource = resource({
    params: () => this.port(),
    loader: ({ params }) => firstValueFrom(params.listEmails()),
  });

  readonly emails = computed(() => this.emailsResource.value() ?? []);
  readonly loading = computed(() => this.emailsResource.isLoading());

  readonly addEmailModel = signal({ email: '' });
  readonly addEmailForm = form(
    this.addEmailModel,
    (p) => {
      required(p.email);
      email(p.email);
    },
    {
      submission: {
        action: async () => {
          const value = this.addEmailModel().email;
          try {
            await firstValueFrom(this.port().addEmail(value));
            this.addEmailModel.set({ email: '' });
            this.addEmailForm().reset();
            this.snackBar.open('Email added', 'OK', { duration: snackSuccessDuration });
            this.emailsResource.reload();
          } catch {
            this.snackBar.open('Failed to add email', 'OK', { duration: snackErrorDuration });
          }
        },
      },
    },
  );

  async removeEmail(emailId: string) {
    try {
      await firstValueFrom(this.port().removeEmail(emailId));
      this.snackBar.open('Email removed', 'OK', { duration: snackSuccessDuration });
      this.emailsResource.reload();
    } catch {
      this.snackBar.open('Failed to remove email', 'OK', { duration: snackErrorDuration });
    }
  }

  async setPrimary(emailId: string) {
    try {
      await firstValueFrom(this.port().setPrimaryEmail(emailId));
      this.snackBar.open('Primary email updated', 'OK', { duration: snackSuccessDuration });
      this.emailsResource.reload();
    } catch {
      this.snackBar.open('Failed to set primary email', 'OK', { duration: snackErrorDuration });
    }
  }
}
