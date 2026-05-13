import { ChangeDetectionStrategy, Component, computed, inject, input, signal } from '@angular/core';
import { form, FormField, FormRoot } from '@angular/forms/signals';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { firstValueFrom } from 'rxjs';
import {
  PasswordConfirmComponent,
  passwordConfirmSchema,
} from '~/frontend/auth/password-confirm/password-confirm.component';
import { snackErrorDuration, snackSuccessDuration } from '~/frontend/constants';
import type { BaseUserPort } from './user-data.port';

@Component({
  selector: 'app-user-password',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    FormField,
    FormRoot,
    MatButtonModule,
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatSnackBarModule,
    PasswordConfirmComponent,
  ],
  templateUrl: './user-password.component.html',
})
export class UserPasswordComponent {
  private readonly snackBar = inject(MatSnackBar);

  readonly port = input.required<BaseUserPort>();
  /** When true the form requires the user's existing password (self-service). */
  readonly requireCurrentPassword = input.required<boolean>();

  readonly model = signal({ currentPassword: '', password: '', confirmPassword: '' });

  readonly passwordForm = form(
    this.model,
    (p) => {
      passwordConfirmSchema(p.password, p.confirmPassword);
    },
    {
      submission: {
        action: async () => {
          const { currentPassword, password } = this.model();
          const requiresCurrent = this.requireCurrentPassword();
          try {
            await firstValueFrom(
              this.port().changePassword({
                currentPassword: requiresCurrent ? currentPassword : undefined,
                password,
              }),
            );
            this.model.set({ currentPassword: '', password: '', confirmPassword: '' });
            this.passwordForm().reset();
            this.snackBar.open(requiresCurrent ? 'Password changed' : 'Password reset', 'OK', {
              duration: snackSuccessDuration,
            });
          } catch {
            this.snackBar.open(
              requiresCurrent
                ? 'Failed to change password. Check your current password.'
                : 'Failed to reset password',
              'OK',
              { duration: snackErrorDuration },
            );
          }
        },
      },
    },
  );

  readonly canSubmit = computed(() => {
    const form = this.passwordForm();
    if (form.submitting() || form.invalid()) return false;
    if (this.requireCurrentPassword() && !this.model().currentPassword) return false;
    return true;
  });
}
