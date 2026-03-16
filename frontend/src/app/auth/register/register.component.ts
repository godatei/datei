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
import { Router, RouterLink } from '@angular/router';
import { AuthService } from '~/frontend/services/auth.service';

function passwordMatchValidator(control: AbstractControl): ValidationErrors | null {
  const password = control.get('password')?.value;
  const confirm = control.get('confirmPassword')?.value;
  return password === confirm ? null : { passwordMismatch: true };
}

@Component({
  selector: 'app-register',
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
            <h1>Create your account</h1>
            <p class="subtitle">to get started with Datei</p>
          </div>

          @if (errorMessage()) {
            <div class="error-banner">{{ errorMessage() }}</div>
          }

          <form class="auth-form" [formGroup]="form" (ngSubmit)="onSubmit()">
            <mat-form-field class="form-field" appearance="outline">
              <mat-label>Name</mat-label>
              <input matInput formControlName="name" autocomplete="name" />
            </mat-form-field>

            <mat-form-field class="form-field" appearance="outline">
              <mat-label>Email</mat-label>
              <input matInput formControlName="email" type="email" autocomplete="email" />
            </mat-form-field>

            <mat-form-field class="form-field" appearance="outline">
              <mat-label>Password</mat-label>
              <input
                matInput
                formControlName="password"
                type="password"
                autocomplete="new-password"
              />
              <mat-hint>At least 8 characters</mat-hint>
            </mat-form-field>

            <mat-form-field class="form-field" appearance="outline">
              <mat-label>Confirm password</mat-label>
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
                Create account
              }
            </button>
          </form>

          <div class="auth-links">
            <a routerLink="/login">Already have an account? Sign in</a>
          </div>
        </mat-card-content>
      </mat-card>
    </div>
  `,
  styleUrls: ['../auth-shared.css'],
})
export class RegisterComponent {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly fb = inject(FormBuilder);

  readonly loading = signal(false);
  readonly errorMessage = signal('');

  readonly form = this.fb.nonNullable.group(
    {
      name: ['', Validators.required],
      email: ['', [Validators.required, Validators.email]],
      password: ['', [Validators.required, Validators.minLength(8)]],
      confirmPassword: ['', Validators.required],
    },
    { validators: passwordMatchValidator },
  );

  onSubmit() {
    if (this.form.invalid) return;
    this.loading.set(true);
    this.errorMessage.set('');

    const { email, name, password } = this.form.getRawValue();
    this.auth.register(email, name, password).subscribe({
      next: () => {
        this.loading.set(false);
        this.router.navigate(['/login'], { queryParams: { email } });
      },
      error: () => {
        this.loading.set(false);
        this.errorMessage.set('Registration failed. Email may already be in use.');
      },
    });
  }
}
