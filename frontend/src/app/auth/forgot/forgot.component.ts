import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { email, form, FormField, required } from '@angular/forms/signals';
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
    FormField,
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
    RouterLink,
  ],
  templateUrl: './forgot.component.html',
  styleUrls: ['../auth-shared.css'],
})
export class ForgotComponent {
  private readonly auth = inject(AuthService);

  readonly loading = signal(false);
  readonly success = signal(false);

  readonly model = signal({ email: '' });
  readonly form = form(this.model, (p) => {
    required(p.email);
    email(p.email);
  });

  onSubmit(event: Event) {
    event.preventDefault();
    if (this.form().invalid()) return;
    this.loading.set(true);

    this.auth.resetPassword(this.model().email).subscribe({
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
