import { Component, inject, input, output, signal } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { Api } from '~/api/api';
import { disableUserMfaAdmin } from '~/api/functions';
import { snackErrorDuration, snackSuccessDuration } from '~/frontend/constants';

@Component({
  selector: 'app-admin-mfa',
  imports: [MatButtonModule, MatCardModule, MatSnackBarModule],
  templateUrl: './admin-mfa.component.html',
})
export class AdminMfaComponent {
  private readonly api = inject(Api);
  private readonly snackBar = inject(MatSnackBar);

  readonly userId = input.required<string>();
  readonly mfaEnabled = input.required<boolean>();
  readonly changed = output<void>();

  readonly loading = signal(false);

  async disable() {
    this.loading.set(true);
    try {
      await this.api.invoke(disableUserMfaAdmin, { id: this.userId() });
      this.snackBar.open('MFA disabled', 'OK', { duration: snackSuccessDuration });
      this.changed.emit();
    } catch {
      this.snackBar.open('Failed to disable MFA', 'OK', { duration: snackErrorDuration });
    } finally {
      this.loading.set(false);
    }
  }
}
