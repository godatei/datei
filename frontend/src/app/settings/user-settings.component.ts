import { Clipboard } from '@angular/cdk/clipboard';
import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { email, form, FormField, minLength, required } from '@angular/forms/signals';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatChipsModule } from '@angular/material/chips';
import { MatDividerModule } from '@angular/material/divider';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import type { UserEmail } from '~/api/models/user-email';
import {
  PasswordConfirmComponent,
  passwordConfirmSchema,
} from '~/frontend/auth/password-confirm/password-confirm.component';
import { AuthService } from '~/frontend/services/auth.service';
import { SettingsService } from '~/frontend/services/settings.service';

@Component({
  selector: 'app-user-settings',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    FormField,
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatChipsModule,
    MatDividerModule,
    MatIconModule,
    MatProgressSpinnerModule,
    MatSnackBarModule,
    PasswordConfirmComponent,
  ],
  templateUrl: './user-settings.component.html',
  styleUrl: './user-settings.component.css',
})
export class UserSettingsComponent {
  private readonly settings = inject(SettingsService);
  private readonly auth = inject(AuthService);
  private readonly snackBar = inject(MatSnackBar);
  private readonly clipboard = inject(Clipboard);

  readonly emails = signal<UserEmail[]>([]);
  readonly emailsLoading = signal(false);
  readonly profileLoading = signal(false);
  readonly passwordLoading = signal(false);
  readonly mfaEnabled = signal(false);
  readonly mfaLoading = signal(false);
  readonly mfaSetupData = signal<{ secret: string; qrCodeUrl: string } | undefined>(undefined);
  readonly recoveryCodes = signal<string[] | undefined>(undefined);

  readonly profileModel = signal({ name: '' });
  readonly profileForm = form(this.profileModel, (p) => {
    required(p.name);
  });

  readonly passwordModel = signal({ currentPassword: '', password: '', confirmPassword: '' });
  readonly passwordForm = form(this.passwordModel, (p) => {
    required(p.currentPassword);
    passwordConfirmSchema(p.password, p.confirmPassword);
  });

  readonly mfaEnableModel = signal({ code: '' });
  readonly mfaEnableForm = form(this.mfaEnableModel, (p) => {
    required(p.code);
    minLength(p.code, 6);
  });

  readonly addEmailModel = signal({ email: '' });
  readonly addEmailForm = form(this.addEmailModel, (p) => {
    required(p.email);
    email(p.email);
  });

  readonly mfaDisableModel = signal({ password: '' });
  readonly mfaDisableForm = form(this.mfaDisableModel, (p) => {
    required(p.password);
  });

  constructor() {
    this.loadProfile();
    this.loadEmails();
  }

  private loadProfile() {
    this.settings.getCurrentUser().subscribe({
      next: (user) => {
        this.profileModel.set({ name: user.name });
        this.mfaEnabled.set(user.mfaEnabled);
      },
    });
  }

  private loadEmails() {
    this.emailsLoading.set(true);
    this.settings.getEmails().subscribe({
      next: (emails) => {
        this.emails.set(emails);
        this.emailsLoading.set(false);
      },
      error: () => this.emailsLoading.set(false),
    });
  }

  addNewEmail(event: Event) {
    event.preventDefault();
    this.emailsLoading.set(true);
    this.settings.addEmail(this.addEmailModel().email).subscribe({
      next: () => {
        this.addEmailModel.set({ email: '' });
        this.snackBar.open('Email added', 'OK', { duration: 3000 });
        this.loadEmails();
      },
      error: () => {
        this.emailsLoading.set(false);
        this.snackBar.open('Failed to add email', 'OK', { duration: 3000 });
      },
    });
  }

  removeEmailAddress(emailId: string) {
    this.emailsLoading.set(true);
    this.settings.removeEmail(emailId).subscribe({
      next: () => {
        this.snackBar.open('Email removed', 'OK', { duration: 3000 });
        this.loadEmails();
      },
      error: () => {
        this.emailsLoading.set(false);
        this.snackBar.open('Failed to remove email', 'OK', { duration: 3000 });
      },
    });
  }

  setPrimary(emailId: string) {
    this.emailsLoading.set(true);
    this.settings.setPrimaryEmail(emailId).subscribe({
      next: () => {
        this.snackBar.open('Primary email updated', 'OK', { duration: 3000 });
        this.loadEmails();
      },
      error: () => {
        this.emailsLoading.set(false);
        this.snackBar.open('Failed to set primary email', 'OK', { duration: 3000 });
      },
    });
  }

  updateProfile(event: Event) {
    event.preventDefault();
    this.profileLoading.set(true);
    this.settings.updateUser({ name: this.profileModel().name }).subscribe({
      next: (res) => {
        this.profileLoading.set(false);
        this.auth.updateName(res.name);
        this.snackBar.open('Profile updated', 'OK', { duration: 3000 });
      },
      error: () => this.profileLoading.set(false),
    });
  }

  changePassword(event: Event) {
    event.preventDefault();
    this.passwordLoading.set(true);
    const { currentPassword, password } = this.passwordModel();
    this.settings.updateUser({ currentPassword, password }).subscribe({
      next: () => {
        this.passwordLoading.set(false);
        this.passwordModel.set({ currentPassword: '', password: '', confirmPassword: '' });
        this.snackBar.open('Password changed', 'OK', { duration: 3000 });
      },
      error: () => this.passwordLoading.set(false),
    });
  }

  setupMFA() {
    this.mfaLoading.set(true);
    this.settings.startMFASetup().subscribe({
      next: (data) => {
        this.mfaLoading.set(false);
        this.mfaSetupData.set(data);
      },
      error: () => this.mfaLoading.set(false),
    });
  }

  enableMFA(event: Event) {
    event.preventDefault();
    this.mfaLoading.set(true);
    this.settings.enableMFA(this.mfaEnableModel().code).subscribe({
      next: (data) => {
        this.mfaLoading.set(false);
        this.mfaEnabled.set(true);
        this.mfaSetupData.set(undefined);
        this.recoveryCodes.set(data.recoveryCodes);
        this.snackBar.open('MFA enabled', 'OK', { duration: 3000 });
      },
      error: () => this.mfaLoading.set(false),
    });
  }

  disableMFA(event: Event) {
    event.preventDefault();
    this.mfaLoading.set(true);
    this.settings.disableMFA(this.mfaDisableModel().password).subscribe({
      next: () => {
        this.mfaLoading.set(false);
        this.mfaEnabled.set(false);
        this.mfaDisableModel.set({ password: '' });
        this.snackBar.open('MFA disabled', 'OK', { duration: 3000 });
      },
      error: () => this.mfaLoading.set(false),
    });
  }

  copySecret() {
    const secret = this.mfaSetupData()?.secret;
    if (secret) {
      this.clipboard.copy(secret);
      this.snackBar.open('Secret copied', 'OK', { duration: 2000 });
    }
  }

  copyRecoveryCodes() {
    const codes = this.recoveryCodes();
    if (codes) {
      this.clipboard.copy(codes.join('\n'));
      this.snackBar.open('Recovery codes copied', 'OK', { duration: 2000 });
    }
  }
}
