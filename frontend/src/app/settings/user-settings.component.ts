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
  template: `
    <div class="settings-container">
      <h2 class="page-title">Settings</h2>

      <!-- Profile -->
      <mat-card>
        <mat-card-header><mat-card-title>Profile</mat-card-title></mat-card-header>
        <mat-card-content>
          <p class="section-description">Manage your display name</p>
          <form [formGroup]="profileForm" (ngSubmit)="updateProfile()">
            <mat-form-field appearance="outline">
              <mat-label>Name</mat-label>
              <input matInput formControlName="name" />
            </mat-form-field>
            <button mat-flat-button type="submit" [disabled]="profileLoading()" class="action-btn">
              Save
            </button>
          </form>
        </mat-card-content>
      </mat-card>

      <!-- Emails -->
      <mat-card>
        <mat-card-header><mat-card-title>Email Addresses</mat-card-title></mat-card-header>
        <mat-card-content>
          <p class="section-description">Control your email addresses and verification status</p>
          @for (email of emails(); track email.id; let last = $last) {
            <div class="email-row">
              <span class="email-address">{{ email.email }}</span>
              <span class="email-badges">
                @if (email.isPrimary) {
                  <mat-chip-set>
                    <mat-chip highlighted>Primary</mat-chip>
                  </mat-chip-set>
                }
                @if (email.verified) {
                  <mat-chip-set>
                    <mat-chip>Verified</mat-chip>
                  </mat-chip-set>
                } @else {
                  <mat-chip-set>
                    <mat-chip class="chip-unverified">Unverified</mat-chip>
                  </mat-chip-set>
                }
              </span>
              <span class="email-actions">
                @if (!email.isPrimary && email.verified) {
                  <button mat-button (click)="setPrimary(email.id)" [disabled]="emailsLoading()">
                    Set primary
                  </button>
                }
                @if (!email.isPrimary) {
                  <button
                    mat-icon-button
                    color="warn"
                    (click)="removeEmailAddress(email.id)"
                    [disabled]="emailsLoading()"
                    [attr.aria-label]="'Remove ' + email.email"
                  >
                    <mat-icon>delete</mat-icon>
                  </button>
                }
              </span>
            </div>
            @if (!last) {
              <mat-divider></mat-divider>
            }
          }
          <form [formGroup]="addEmailForm" (ngSubmit)="addNewEmail()" class="add-email-form">
            <mat-form-field appearance="outline">
              <mat-label>Add email address</mat-label>
              <input matInput formControlName="email" type="email" />
            </mat-form-field>
            <button
              mat-flat-button
              type="submit"
              [disabled]="emailsLoading() || addEmailForm.invalid"
            >
              Add email
            </button>
          </form>
        </mat-card-content>
      </mat-card>

      <!-- Change Password -->
      <mat-card>
        <mat-card-header><mat-card-title>Change Password</mat-card-title></mat-card-header>
        <mat-card-content>
          <p class="section-description">Update your password</p>
          <form [formGroup]="passwordForm" (ngSubmit)="changePassword()">
            <mat-form-field appearance="outline">
              <mat-label>Current password</mat-label>
              <input
                matInput
                formControlName="currentPassword"
                type="password"
                autocomplete="current-password"
              />
            </mat-form-field>
            <mat-form-field appearance="outline">
              <mat-label>New password</mat-label>
              <input
                matInput
                formControlName="password"
                type="password"
                autocomplete="new-password"
              />
              <mat-hint>At least 8 characters</mat-hint>
            </mat-form-field>
            <mat-form-field appearance="outline">
              <mat-label>Confirm new password</mat-label>
              <input
                matInput
                formControlName="confirmPassword"
                type="password"
                autocomplete="new-password"
              />
              @if (passwordForm.hasError('passwordMismatch')) {
                <mat-error>Passwords do not match</mat-error>
              }
            </mat-form-field>
            <button
              mat-flat-button
              type="submit"
              [disabled]="passwordLoading() || passwordForm.invalid"
            >
              Change password
            </button>
          </form>
        </mat-card-content>
      </mat-card>

      <!-- MFA -->
      <mat-card>
        <mat-card-header
          ><mat-card-title>Two-Factor Authentication</mat-card-title></mat-card-header
        >
        <mat-card-content>
          <p class="section-description">Add an extra layer of security to your account</p>

          @if (mfaSetupData()) {
            <div class="mfa-setup">
              <p>Scan this QR code with your authenticator app:</p>
              <div class="qr-container">
                <img [src]="mfaSetupData()!.qrCodeUrl" alt="MFA QR Code" width="200" height="200" />
              </div>
              <div class="secret-row">
                <span class="mfa-secret">{{ mfaSetupData()!.secret }}</span>
                <button
                  mat-icon-button
                  (click)="copySecret()"
                  aria-label="Copy secret to clipboard"
                >
                  <mat-icon>content_copy</mat-icon>
                </button>
              </div>
              <form [formGroup]="mfaEnableForm" (ngSubmit)="enableMFA()">
                <mat-form-field appearance="outline">
                  <mat-label>Verification code</mat-label>
                  <input matInput formControlName="code" autocomplete="one-time-code" />
                </mat-form-field>
                <button
                  mat-flat-button
                  type="submit"
                  [disabled]="mfaLoading() || mfaEnableForm.invalid"
                >
                  Enable MFA
                </button>
              </form>
            </div>
          } @else if (recoveryCodes()) {
            <p>Save these recovery codes in a safe place:</p>
            <div class="recovery-codes-container">
              <ul class="recovery-codes" role="list" aria-label="Recovery codes">
                @for (code of recoveryCodes()!; track code) {
                  <li>{{ code }}</li>
                }
              </ul>
              <button mat-button (click)="copyRecoveryCodes()">
                <mat-icon>content_copy</mat-icon>
                Copy all
              </button>
            </div>
            <button mat-flat-button (click)="recoveryCodes.set(undefined)">Done</button>
          } @else {
            <button
              mat-flat-button
              (click)="setupMFA()"
              [disabled]="mfaLoading()"
              class="action-btn"
            >
              Set up MFA
            </button>

            <div class="danger-zone">
              <p class="danger-label">Disable two-factor authentication</p>
              <form [formGroup]="mfaDisableForm" (ngSubmit)="disableMFA()">
                <mat-form-field appearance="outline">
                  <mat-label>Password</mat-label>
                  <input matInput formControlName="password" type="password" />
                </mat-form-field>
                <button
                  mat-button
                  color="warn"
                  type="submit"
                  [disabled]="mfaLoading() || mfaDisableForm.invalid"
                >
                  Disable MFA
                </button>
              </form>
            </div>
          }
        </mat-card-content>
      </mat-card>
    </div>
  `,
  styles: `
    .settings-container {
      max-width: 600px;
      margin: 2rem auto;
      display: flex;
      flex-direction: column;
      gap: 1.5rem;
      padding: 0 1rem;
    }
    .page-title {
      font: var(--mat-sys-headline-small);
      margin: 0;
    }
    .section-description {
      font: var(--mat-sys-body-medium);
      color: var(--mat-sys-on-surface-variant);
      margin: 0 0 1rem;
    }
    form {
      display: flex;
      flex-direction: column;
      gap: 0.5rem;
    }
    mat-form-field {
      width: 100%;
    }
    .action-btn {
      align-self: flex-end;
    }
    .email-row {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      padding: 0.75rem 0;
    }
    .email-address {
      flex: 1;
      word-break: break-all;
    }
    .email-badges {
      display: flex;
      gap: 0.25rem;
    }
    .email-actions {
      display: flex;
      gap: 0.25rem;
    }
    .chip-unverified {
      opacity: 0.7;
    }
    .add-email-form {
      margin-top: 1rem;
    }
    .mfa-setup {
      display: flex;
      flex-direction: column;
      gap: 0.75rem;
    }
    .qr-container {
      display: flex;
      justify-content: center;
      background: var(--mat-sys-surface-container);
      border-radius: 12px;
      padding: 1rem;
    }
    .secret-row {
      display: flex;
      align-items: center;
      gap: 0.5rem;
    }
    .mfa-secret {
      font-family: monospace;
      word-break: break-all;
      flex: 1;
    }
    .recovery-codes-container {
      background: var(--mat-sys-surface-container);
      border-radius: 8px;
      padding: 1rem;
    }
    .recovery-codes {
      font-family: monospace;
      list-style: none;
      padding: 0;
      margin: 0 0 0.5rem;
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 0.25rem;
    }
    .danger-zone {
      border: 1px solid var(--mat-sys-error);
      border-radius: 8px;
      padding: 1rem;
      margin-top: 1rem;
    }
    .danger-label {
      font: var(--mat-sys-label-large);
      color: var(--mat-sys-error);
      margin: 0 0 0.75rem;
    }
  `,
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
    name: [this.auth.getClaims()?.name ?? '', Validators.required],
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
    this.loadEmails();
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
      next: () => {
        this.profileLoading.set(false);
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
