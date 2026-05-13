import { ChangeDetectionStrategy, Component, inject, input, OnInit, signal } from '@angular/core';
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
import type { UserEmail } from '~/api/models/user-email';
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
export class UserEmailsComponent implements OnInit {
  private readonly snackBar = inject(MatSnackBar);

  readonly port = input.required<BaseUserPort>();

  readonly emails = signal<UserEmail[]>([]);
  readonly loading = signal(false);

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
            this.loadEmails();
          } catch {
            this.snackBar.open('Failed to add email', 'OK', { duration: snackErrorDuration });
          }
        },
      },
    },
  );

  ngOnInit() {
    this.loadEmails();
  }

  removeEmail(emailId: string) {
    this.loading.set(true);
    this.port()
      .removeEmail(emailId)
      .subscribe({
        next: () => {
          this.snackBar.open('Email removed', 'OK', { duration: snackSuccessDuration });
          this.loadEmails();
        },
        error: () => {
          this.loading.set(false);
          this.snackBar.open('Failed to remove email', 'OK', { duration: snackErrorDuration });
        },
      });
  }

  setPrimary(emailId: string) {
    this.loading.set(true);
    this.port()
      .setPrimaryEmail(emailId)
      .subscribe({
        next: () => {
          this.snackBar.open('Primary email updated', 'OK', { duration: snackSuccessDuration });
          this.loadEmails();
        },
        error: () => {
          this.loading.set(false);
          this.snackBar.open('Failed to set primary email', 'OK', { duration: snackErrorDuration });
        },
      });
  }

  private loadEmails() {
    this.loading.set(true);
    this.port()
      .listEmails()
      .subscribe({
        next: (emails) => {
          this.emails.set(emails);
          this.loading.set(false);
        },
        error: () => this.loading.set(false),
      });
  }
}
