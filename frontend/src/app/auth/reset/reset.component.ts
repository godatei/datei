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
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { Router } from '@angular/router';
import { AuthService } from '~/frontend/services/auth.service';
import { SettingsService } from '~/frontend/services/settings.service';

function passwordMatchValidator(control: AbstractControl): ValidationErrors | null {
  const password = control.get('password')?.value;
  const confirm = control.get('confirmPassword')?.value;
  return password === confirm ? null : { passwordMismatch: true };
}

@Component({
  selector: 'app-reset',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    ReactiveFormsModule,
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
  ],
  template: `
    <div class="auth-container">
      <mat-card class="auth-card">
        <mat-card-content>
          <div class="auth-header">
            <mat-icon class="brand-icon">cloud_upload</mat-icon>
            <h1>Set a new password</h1>
            <p class="subtitle">Choose a strong password for your account</p>
          </div>

          @if (errorMessage()) {
            <div class="error-banner">{{ errorMessage() }}</div>
          }

          <form class="auth-form" [formGroup]="form" (ngSubmit)="onSubmit()">
            <mat-form-field class="form-field" appearance="outline">
              <mat-label>New password</mat-label>
              <input
                matInput
                formControlName="password"
                type="password"
                autocomplete="new-password"
              />
              <mat-hint>At least 8 characters</mat-hint>
            </mat-form-field>

            <mat-form-field class="form-field" appearance="outline">
              <mat-label>Confirm new password</mat-label>
              <input
                matInput
                formControlName="confirmPassword"
                type="password"
                autocomplete="new-password"
              />
              @if (form.hasError('passwordMismatch')) {
                <mat-error>Passwords do not match</mat-error>
              }
            </mat-form-field>

            <button
              class="submit-btn"
              mat-flat-button
              type="submit"
              [disabled]="loading() || form.invalid"
            >
              @if (loading()) {
                <mat-spinner diameter="20"></mat-spinner>
              } @else {
                Reset password
              }
            </button>
          </form>
        </mat-card-content>
      </mat-card>
    </div>
  `,
  styleUrls: ['../auth-shared.css'],
})
export class ResetComponent {
  private readonly settings = inject(SettingsService);
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly fb = inject(FormBuilder);

  readonly loading = signal(false);
  readonly errorMessage = signal('');

  readonly form = this.fb.nonNullable.group(
    {
      password: ['', [Validators.required, Validators.minLength(8)]],
      confirmPassword: ['', Validators.required],
    },
    { validators: passwordMatchValidator },
  );

  onSubmit() {
    if (this.form.invalid) return;
    this.loading.set(true);
    this.errorMessage.set('');

    this.settings.updateUser({ password: this.form.getRawValue().password }, true).subscribe({
      next: () => {
        this.loading.set(false);
        this.auth.logout();
        this.router.navigate(['/login']);
      },
      error: () => {
        this.loading.set(false);
        this.errorMessage.set('Password reset failed');
      },
    });
  }
}
