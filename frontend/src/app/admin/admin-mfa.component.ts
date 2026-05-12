import { ChangeDetectionStrategy, Component, inject, input, output, signal } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { snackErrorDuration, snackSuccessDuration } from '~/frontend/constants';
import { AdminUsersService } from '~/frontend/services/admin-users.service';

@Component({
  selector: 'app-admin-mfa',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [MatButtonModule, MatCardModule, MatSnackBarModule],
  templateUrl: './admin-mfa.component.html',
})
export class AdminMfaComponent {
  private readonly admin = inject(AdminUsersService);
  private readonly snackBar = inject(MatSnackBar);

  readonly userId = input.required<string>();
  readonly mfaEnabled = input.required<boolean>();
  readonly changed = output<void>();

  readonly loading = signal(false);

  disable() {
    this.loading.set(true);
    this.admin.disableMfa(this.userId()).subscribe({
      next: () => {
        this.loading.set(false);
        this.snackBar.open('MFA disabled', 'OK', { duration: snackSuccessDuration });
        this.changed.emit();
      },
      error: () => {
        this.loading.set(false);
        this.snackBar.open('Failed to disable MFA', 'OK', { duration: snackErrorDuration });
      },
    });
  }
}
