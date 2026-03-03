import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { AuthService } from '~/frontend/services/auth.service';
import { SettingsService } from '~/frontend/services/settings.service';

@Component({
  selector: 'app-user-settings',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    ReactiveFormsModule,
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatProgressSpinnerModule,
    MatSnackBarModule,
  ],
  template: `
    <div class="settings-container">
      <h2>Settings</h2>

      <!-- Profile -->
      <mat-card>
        <mat-card-header><mat-card-title>Profile</mat-card-title></mat-card-header>
        <mat-card-content>
          <form [formGroup]="profileForm" (ngSubmit)="updateProfile()">
            <mat-form-field appearance="outline">
              <mat-label>Name</mat-label>
              <input matInput formControlName="name" />
            </mat-form-field>
            <button mat-flat-button type="submit" [disabled]="profileLoading()">Save</button>
          </form>
        </mat-card-content>
      </mat-card>

      <!-- Change Password -->
      <mat-card>
        <mat-card-header><mat-card-title>Change Password</mat-card-title></mat-card-header>
        <mat-card-content>
          <form [formGroup]="passwordForm" (ngSubmit)="changePassword()">
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
          @if (mfaSetupData()) {
            <p>Scan this QR code with your authenticator app:</p>
            <img [src]="mfaSetupData()!.qrCodeUrl" alt="MFA QR Code" width="200" height="200" />
            <p class="mfa-secret">Secret: {{ mfaSetupData()!.secret }}</p>
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
          } @else if (recoveryCodes()) {
            <p>Save these recovery codes in a safe place:</p>
            <pre class="recovery-codes">{{
              recoveryCodes()!.join(
                '
'
              )
            }}</pre>
            <button mat-flat-button (click)="recoveryCodes.set(undefined)">Done</button>
          } @else {
            <button mat-flat-button (click)="setupMFA()" [disabled]="mfaLoading()">
              Set up MFA
            </button>
            <form [formGroup]="mfaDisableForm" (ngSubmit)="disableMFA()">
              <mat-form-field appearance="outline">
                <mat-label>Password (to disable MFA)</mat-label>
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
    form {
      display: flex;
      flex-direction: column;
      gap: 0.5rem;
    }
    mat-form-field {
      width: 100%;
    }
    .mfa-secret {
      font-family: monospace;
      word-break: break-all;
    }
    .recovery-codes {
      font-family: monospace;
      background: var(--mat-sys-surface-container);
      padding: 1rem;
      border-radius: 8px;
    }
  `,
})
export class UserSettingsComponent {
  private readonly settings = inject(SettingsService);
  private readonly auth = inject(AuthService);
  private readonly snackBar = inject(MatSnackBar);
  private readonly fb = inject(FormBuilder);

  readonly profileLoading = signal(false);
  readonly passwordLoading = signal(false);
  readonly mfaLoading = signal(false);
  readonly mfaSetupData = signal<{ secret: string; qrCodeUrl: string } | undefined>(undefined);
  readonly recoveryCodes = signal<string[] | undefined>(undefined);

  readonly profileForm = this.fb.nonNullable.group({
    name: [this.auth.getClaims()?.name ?? '', Validators.required],
  });

  readonly passwordForm = this.fb.nonNullable.group({
    password: ['', [Validators.required, Validators.minLength(8)]],
  });

  readonly mfaEnableForm = this.fb.nonNullable.group({
    code: ['', [Validators.required, Validators.minLength(6)]],
  });

  readonly mfaDisableForm = this.fb.nonNullable.group({
    password: ['', Validators.required],
  });

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
    this.settings.updateUser({ password: this.passwordForm.getRawValue().password }).subscribe({
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
}
