import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { email, form, FormField, FormRoot, minLength, required } from '@angular/forms/signals';
import { NgOptimizedImage } from '@angular/common';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { Router, RouterLink, ActivatedRoute } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { AuthService } from '~/frontend/services/auth.service';

@Component({
  selector: 'app-login',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    NgOptimizedImage,
    FormField,
    FormRoot,
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
    RouterLink,
  ],
  templateUrl: './login.component.html',
  host: { class: 'block' },
})
export class LoginComponent {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);

  readonly errorMessage = signal('');
  readonly mfaRequired = signal(false);

  readonly loginModel = signal({ email: '', password: '' });
  readonly loginForm = form(
    this.loginModel,
    (p) => {
      required(p.email);
      email(p.email);
      required(p.password);
    },
    {
      submission: {
        action: async () => {
          this.errorMessage.set('');
          const { email, password } = this.loginModel();
          try {
            const result = await firstValueFrom(this.auth.login(email, password));
            if (result.requiresMfa) {
              this.mfaRequired.set(true);
            } else {
              this.router.navigate(['/']);
            }
          } catch {
            this.errorMessage.set('Invalid email or password');
          }
        },
      },
    },
  );

  readonly mfaModel = signal({ code: '' });
  readonly mfaForm = form(
    this.mfaModel,
    (p) => {
      required(p.code);
      minLength(p.code, 6);
    },
    {
      submission: {
        action: async () => {
          this.errorMessage.set('');
          const { email, password } = this.loginModel();
          const { code } = this.mfaModel();
          try {
            await firstValueFrom(this.auth.login(email, password, code));
            this.router.navigate(['/']);
          } catch {
            this.errorMessage.set('Invalid MFA code');
          }
        },
      },
    },
  );

  constructor() {
    const emailParam = this.route.snapshot.queryParamMap.get('email');
    if (emailParam) {
      this.loginModel.update((m) => ({ ...m, email: emailParam }));
    }
  }
}
