import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { email, form, FormField, minLength, required } from '@angular/forms/signals';
import { NgOptimizedImage } from '@angular/common';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { Router, RouterLink, ActivatedRoute } from '@angular/router';
import { AuthService } from '~/frontend/services/auth.service';

@Component({
  selector: 'app-login',
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
    RouterLink,
  ],
  templateUrl: './login.component.html',
  styleUrls: ['../auth-shared.css'],
})
export class LoginComponent {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);

  readonly loading = signal(false);
  readonly errorMessage = signal('');
  readonly mfaRequired = signal(false);

  readonly loginModel = signal({ email: '', password: '' });
  readonly loginForm = form(this.loginModel, (p) => {
    required(p.email);
    email(p.email);
    required(p.password);
  });

  readonly mfaModel = signal({ code: '' });
  readonly mfaForm = form(this.mfaModel, (p) => {
    required(p.code);
    minLength(p.code, 6);
  });

  constructor() {
    const emailParam = this.route.snapshot.queryParamMap.get('email');
    if (emailParam) {
      this.loginModel.update((m) => ({ ...m, email: emailParam }));
    }
  }

  onSubmit(event: Event) {
    event.preventDefault();
    if (this.loginForm().invalid()) return;
    this.loading.set(true);
    this.errorMessage.set('');

    const { email, password } = this.loginModel();
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

  onMFASubmit(event: Event) {
    event.preventDefault();
    if (this.mfaForm().invalid()) return;
    this.loading.set(true);
    this.errorMessage.set('');

    const { email, password } = this.loginModel();
    const { code } = this.mfaModel();
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
