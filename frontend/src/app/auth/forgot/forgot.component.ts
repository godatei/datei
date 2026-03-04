import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
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
    MatProgressSpinnerModule,
    RouterLink,
  ],
  template: `
    <div class="auth-container">
      <mat-card>
        <mat-card-header>
          <mat-card-title>Reset your password</mat-card-title>
        </mat-card-header>
        <mat-card-content>
          @if (success()) {
            <p>
              If an account exists with that email, you will receive a password reset link shortly.
            </p>
            <a routerLink="/login" mat-flat-button>Back to login</a>
          } @else {
            <form [formGroup]="form" (ngSubmit)="onSubmit()">
              <mat-form-field appearance="outline">
                <mat-label>Email</mat-label>
                <input matInput formControlName="email" type="email" autocomplete="email" />
              </mat-form-field>

              <button mat-flat-button type="submit" [disabled]="loading() || form.invalid">
                @if (loading()) {
                  <mat-spinner diameter="20"></mat-spinner>
                } @else {
                  Send reset link
                }
              </button>
            </form>

            <div class="auth-links">
              <a routerLink="/login">Back to login</a>
            </div>
          }
        </mat-card-content>
      </mat-card>
    </div>
  `,
  styles: `
    .auth-container {
      display: flex;
      justify-content: center;
      align-items: center;
      min-height: 100vh;
      padding: 1rem;
    }
    mat-card {
      max-width: 400px;
      width: 100%;
    }
    form {
      display: flex;
      flex-direction: column;
      gap: 0.5rem;
    }
    mat-form-field {
      width: 100%;
    }
    .auth-links {
      display: flex;
      justify-content: center;
      margin-top: 1rem;
    }
    .auth-links a {
      color: var(--mat-sys-primary);
    }
  `,
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
