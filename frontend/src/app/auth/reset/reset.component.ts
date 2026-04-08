import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { Router } from '@angular/router';
import { AuthService } from '~/frontend/services/auth.service';
import { SettingsService } from '~/frontend/services/settings.service';
import {
  PasswordConfirmComponent,
  passwordConfirmControls,
  passwordMatchValidator,
} from '../password-confirm/password-confirm.component';

@Component({
  selector: 'app-reset',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    ReactiveFormsModule,
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
  private readonly fb = inject(FormBuilder);

  readonly loading = signal(false);
  readonly errorMessage = signal('');

  readonly form = this.fb.nonNullable.group(
    { ...passwordConfirmControls() },
    { validators: passwordMatchValidator },
  );

  onSubmit() {
    if (this.form.invalid) return;
    this.loading.set(true);
    this.errorMessage.set('');

    this.settings.updateUser({ password: this.form.getRawValue().password }, true).subscribe({
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
