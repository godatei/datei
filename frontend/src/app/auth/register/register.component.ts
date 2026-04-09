import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { email, form, FormField, required } from '@angular/forms/signals';
import { NgOptimizedImage } from '@angular/common';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { Router, RouterLink } from '@angular/router';
import { AuthService } from '~/frontend/services/auth.service';
import {
  PasswordConfirmComponent,
  passwordConfirmSchema,
} from '../password-confirm/password-confirm.component';

@Component({
  selector: 'app-register',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    NgOptimizedImage,
    FormField,
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
    PasswordConfirmComponent,
    RouterLink,
  ],
  templateUrl: './register.component.html',
  styleUrls: ['../auth-shared.css'],
})
export class RegisterComponent {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);

  readonly loading = signal(false);
  readonly errorMessage = signal('');

  readonly model = signal({ name: '', email: '', password: '', confirmPassword: '' });
  readonly form = form(this.model, (p) => {
    required(p.name);
    required(p.email);
    email(p.email);
    passwordConfirmSchema(p.password, p.confirmPassword);
  });

  onSubmit(event: Event) {
    event.preventDefault();
    if (this.form().invalid()) return;
    this.loading.set(true);
    this.errorMessage.set('');

    const { email, name, password } = this.model();
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
