import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { Router, RouterLink, ActivatedRoute } from '@angular/router';
import { AuthService } from '~/frontend/services/auth.service';

@Component({
  selector: 'app-login',
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
          <mat-card-title>Sign in to Datei</mat-card-title>
        </mat-card-header>
        <mat-card-content>
          @if (errorMessage()) {
            <p class="error-message">{{ errorMessage() }}</p>
          }

          @if (!mfaRequired()) {
            <form [formGroup]="loginForm" (ngSubmit)="onSubmit()">
              <mat-form-field appearance="outline">
                <mat-label>Email</mat-label>
                <input matInput formControlName="email" type="email" autocomplete="email" />
              </mat-form-field>

              <mat-form-field appearance="outline">
                <mat-label>Password</mat-label>
                <input
                  matInput
                  formControlName="password"
                  type="password"
                  autocomplete="current-password"
                />
              </mat-form-field>

              <button mat-flat-button type="submit" [disabled]="loading() || loginForm.invalid">
                @if (loading()) {
                  <mat-spinner diameter="20"></mat-spinner>
                } @else {
                  Sign in
                }
              </button>
            </form>
          } @else {
            <form [formGroup]="mfaForm" (ngSubmit)="onMFASubmit()">
              <mat-form-field appearance="outline">
                <mat-label>MFA Code</mat-label>
                <input matInput formControlName="code" autocomplete="one-time-code" />
                <mat-hint>Enter your 6-digit code or a recovery code</mat-hint>
              </mat-form-field>

              <button mat-flat-button type="submit" [disabled]="loading() || mfaForm.invalid">
                @if (loading()) {
                  <mat-spinner diameter="20"></mat-spinner>
                } @else {
                  Verify
                }
              </button>
            </form>
          }

          <div class="auth-links">
            <a routerLink="/forgot">Forgot password?</a>
            <a routerLink="/register">Create an account</a>
          </div>
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
    .error-message {
      color: var(--mat-sys-error);
      margin-bottom: 1rem;
    }
    .auth-links {
      display: flex;
      justify-content: space-between;
      margin-top: 1rem;
    }
    .auth-links a {
      color: var(--mat-sys-primary);
    }
  `,
})
export class LoginComponent {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);
  private readonly fb = inject(FormBuilder);

  readonly loading = signal(false);
  readonly errorMessage = signal('');
  readonly mfaRequired = signal(false);

  readonly loginForm = this.fb.nonNullable.group({
    email: ['', [Validators.required, Validators.email]],
    password: ['', Validators.required],
  });

  readonly mfaForm = this.fb.nonNullable.group({
    code: ['', [Validators.required, Validators.minLength(6)]],
  });

  constructor() {
    const email = this.route.snapshot.queryParamMap.get('email');
    if (email) {
      this.loginForm.patchValue({ email });
    }
  }

  onSubmit() {
    if (this.loginForm.invalid) return;
    this.loading.set(true);
    this.errorMessage.set('');

    const { email, password } = this.loginForm.getRawValue();
    this.auth.login(email, password).subscribe({
      next: (result) => {
        this.loading.set(false);
        if (result.requiresMfa) {
          this.mfaRequired.set(true);
        } else {
          this.router.navigate(['/']);
        }
      },
      error: () => {
        this.loading.set(false);
        this.errorMessage.set('Invalid email or password');
      },
    });
  }

  onMFASubmit() {
    if (this.mfaForm.invalid) return;
    this.loading.set(true);
    this.errorMessage.set('');

    const { email, password } = this.loginForm.getRawValue();
    const { code } = this.mfaForm.getRawValue();
    this.auth.login(email, password, code).subscribe({
      next: () => {
        this.loading.set(false);
        this.router.navigate(['/']);
      },
      error: () => {
        this.loading.set(false);
        this.errorMessage.set('Invalid MFA code');
      },
    });
  }
}
