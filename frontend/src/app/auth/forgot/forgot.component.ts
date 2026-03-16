import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { RouterLink } from '@angular/router';
import { AuthService } from '~/frontend/services/auth.service';

@Component({
  selector: 'app-forgot',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    ReactiveFormsModule,
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
    RouterLink,
  ],
  template: `
    <div class="auth-container">
      <mat-card class="auth-card">
        <mat-card-content>
          <div class="auth-header">
            <mat-icon class="brand-icon">cloud_upload</mat-icon>
            <h1>Reset your password</h1>
            <p class="subtitle">We'll send you a reset link</p>
          </div>

          @if (success()) {
            <p class="success-message">
              If an account exists with that email, you will receive a password reset link shortly.
            </p>
            <a class="submit-btn" routerLink="/login" mat-flat-button>Back to sign in</a>
          } @else {
            <form class="auth-form" [formGroup]="form" (ngSubmit)="onSubmit()">
              <mat-form-field class="form-field" appearance="outline">
                <mat-label>Email</mat-label>
                <input matInput formControlName="email" type="email" autocomplete="email" />
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
                  Send reset link
                }
              </button>
            </form>

            <div class="auth-links">
              <a routerLink="/login">Back to sign in</a>
            </div>
          }
        </mat-card-content>
      </mat-card>
    </div>
  `,
  styleUrls: ['../auth-shared.css'],
})
export class ForgotComponent {
  private readonly auth = inject(AuthService);
  private readonly fb = inject(FormBuilder);

  readonly loading = signal(false);
  readonly success = signal(false);

  readonly form = this.fb.nonNullable.group({
    email: ['', [Validators.required, Validators.email]],
  });

  onSubmit() {
    if (this.form.invalid) return;
    this.loading.set(true);

    this.auth.resetPassword(this.form.getRawValue().email).subscribe({
      next: () => {
        this.loading.set(false);
        this.success.set(true);
      },
      error: () => {
        this.loading.set(false);
        this.success.set(true); // Don't reveal if email exists
      },
    });
  }
}
