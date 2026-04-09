import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { form } from '@angular/forms/signals';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { Router } from '@angular/router';
import { AuthService } from '~/frontend/services/auth.service';
import { SettingsService } from '~/frontend/services/settings.service';
import {
  PasswordConfirmComponent,
  passwordConfirmSchema,
} from '../password-confirm/password-confirm.component';

@Component({
  selector: 'app-reset',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
    PasswordConfirmComponent,
  ],
  templateUrl: './reset.component.html',
  styleUrls: ['../auth-shared.css'],
})
export class ResetComponent {
  private readonly settings = inject(SettingsService);
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);

  readonly loading = signal(false);
  readonly errorMessage = signal('');

  readonly model = signal({ password: '', confirmPassword: '' });
  readonly form = form(this.model, (p) => {
    passwordConfirmSchema(p.password, p.confirmPassword);
  });

  onSubmit(event: Event) {
    event.preventDefault();
    if (this.form().invalid()) return;
    this.loading.set(true);
    this.errorMessage.set('');

    this.settings.updateUser({ password: this.model().password }, true).subscribe({
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
