import { Clipboard } from '@angular/cdk/clipboard';
import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import {
  AbstractControl,
  FormBuilder,
  ReactiveFormsModule,
  ValidationErrors,
  Validators,
} from '@angular/forms';
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
import { AuthService } from '~/frontend/services/auth.service';
import { SettingsService } from '~/frontend/services/settings.service';

function passwordMatchValidator(control: AbstractControl): ValidationErrors | null {
  const password = control.get('password')?.value;
  const confirm = control.get('confirmPassword')?.value;
  return password === confirm ? null : { passwordMismatch: true };
}

@Component({
  selector: 'app-user-settings',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    ReactiveFormsModule,
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatChipsModule,
    MatDividerModule,
    MatIconModule,
    MatProgressSpinnerModule,
    MatSnackBarModule,
  ],
  templateUrl: './user-settings.component.html',
  styleUrl: './user-settings.component.css',
})
export class UserSettingsComponent {
  private readonly settings = inject(SettingsService);
  private readonly auth = inject(AuthService);
  private readonly snackBar = inject(MatSnackBar);
  private readonly clipboard = inject(Clipboard);
  private readonly fb = inject(FormBuilder);

  readonly emails = signal<UserEmail[]>([]);
  readonly emailsLoading = signal(false);
  readonly profileLoading = signal(false);
  readonly passwordLoading = signal(false);
  readonly mfaLoading = signal(false);
  readonly mfaSetupData = signal<{ secret: string; qrCodeUrl: string } | undefined>(undefined);
  readonly recoveryCodes = signal<string[] | undefined>(undefined);

  readonly profileForm = this.fb.nonNullable.group({
    name: ['', Validators.required],
  });

  readonly passwordForm = this.fb.nonNullable.group(
    {
      currentPassword: ['', Validators.required],
      password: ['', [Validators.required, Validators.minLength(8)]],
      confirmPassword: ['', Validators.required],
    },
    { validators: passwordMatchValidator },
  );

  readonly mfaEnableForm = this.fb.nonNullable.group({
    code: ['', [Validators.required, Validators.minLength(6)]],
  });

  readonly addEmailForm = this.fb.nonNullable.group({
    email: ['', [Validators.required, Validators.email]],
  });

  readonly mfaDisableForm = this.fb.nonNullable.group({
    password: ['', Validators.required],
  });

  constructor() {
    this.loadProfile();
    this.loadEmails();
  }

  private loadProfile() {
    this.settings.getCurrentUser().subscribe({
      next: (user) => this.profileForm.patchValue({ name: user.name }),
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

  addNewEmail() {
    this.emailsLoading.set(true);
    this.settings.addEmail(this.addEmailForm.getRawValue().email).subscribe({
      next: () => {
        this.addEmailForm.reset();
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

  updateProfile() {
    this.profileLoading.set(true);
    this.settings.updateUser({ name: this.profileForm.getRawValue().name }).subscribe({
      next: (res) => {
        this.profileLoading.set(false);
        this.auth.updateName(res.name);
        this.snackBar.open('Profile updated', 'OK', { duration: 3000 });
      },
      error: () => this.profileLoading.set(false),
    });
  }

  changePassword() {
    this.passwordLoading.set(true);
    const { currentPassword, password } = this.passwordForm.getRawValue();
    this.settings.updateUser({ currentPassword, password }).subscribe({
      next: () => {
        this.passwordLoading.set(false);
        this.passwordForm.reset();
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

  enableMFA() {
    this.mfaLoading.set(true);
    this.settings.enableMFA(this.mfaEnableForm.getRawValue().code).subscribe({
      next: (data) => {
        this.mfaLoading.set(false);
        this.mfaSetupData.set(undefined);
        this.recoveryCodes.set(data.recoveryCodes);
        this.snackBar.open('MFA enabled', 'OK', { duration: 3000 });
      },
      error: () => this.mfaLoading.set(false),
    });
  }

  disableMFA() {
    this.mfaLoading.set(true);
    this.settings.disableMFA(this.mfaDisableForm.getRawValue().password).subscribe({
      next: () => {
        this.mfaLoading.set(false);
        this.mfaDisableForm.reset();
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
