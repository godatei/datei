import { Component, inject, signal } from '@angular/core';
import { email, form, FormField, FormRoot, required } from '@angular/forms/signals';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { AuthService } from '~/frontend/services/auth.service';
import { AuthLayoutComponent } from '../auth-layout/auth-layout.component';

@Component({
  selector: 'app-forgot',
  imports: [
    FormField,
    FormRoot,
    AuthLayoutComponent,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatProgressSpinnerModule,
    RouterLink,
  ],
  templateUrl: './forgot.component.html',
  host: { class: 'block' },
})
export class ForgotComponent {
  private readonly auth = inject(AuthService);

  readonly success = signal(false);

  readonly model = signal({ email: '' });
  readonly form = form(
    this.model,
    (p) => {
      required(p.email);
      email(p.email);
    },
    {
      submission: {
        action: async () => {
          try {
            await firstValueFrom(this.auth.resetPassword(this.model().email));
          } catch {
            // Don't reveal if email exists
          }
          this.success.set(true);
        },
      },
    },
  );
}
