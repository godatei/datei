import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { form, FormRoot } from '@angular/forms/signals';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { Router } from '@angular/router';
import { firstValueFrom } from 'rxjs';
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
    FormRoot,
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

  readonly errorMessage = signal('');

  readonly model = signal({ password: '', confirmPassword: '' });
  readonly form = form(
    this.model,
    (p) => {
      passwordConfirmSchema(p.password, p.confirmPassword);
    },
    {
      submission: {
        action: async () => {
          this.errorMessage.set('');
          try {
            await firstValueFrom(
              this.settings.updateUser({ password: this.model().password }, true),
            );
            this.auth.logout();
            this.router.navigate(['/login']);
          } catch {
            this.errorMessage.set('Password reset failed');
          }
        },
      },
    },
  );
}
