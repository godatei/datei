import {
  ChangeDetectionStrategy,
  Component,
  effect,
  inject,
  input,
  output,
  signal,
} from '@angular/core';
import { MatCardModule } from '@angular/material/card';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { snackErrorDuration, snackSuccessDuration } from '~/frontend/constants';
import { AdminUsersService } from '~/frontend/services/admin-users.service';

@Component({
  selector: 'app-admin-role',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [MatCardModule, MatSlideToggleModule, MatSnackBarModule],
  templateUrl: './admin-role.component.html',
})
export class AdminRoleComponent {
  private readonly admin = inject(AdminUsersService);
  private readonly snackBar = inject(MatSnackBar);

  readonly userId = input.required<string>();
  readonly isAdmin = input.required<boolean>();
  readonly changed = output<void>();

  readonly checked = signal(false);
  readonly loading = signal(false);

  constructor() {
    effect(() => {
      this.checked.set(this.isAdmin());
    });
  }

  toggle(value: boolean) {
    const previous = this.checked();
    this.checked.set(value);
    this.loading.set(true);
    this.admin.updateUser(this.userId(), { isAdmin: value }).subscribe({
      next: () => {
        this.loading.set(false);
        this.snackBar.open(
          value ? 'Granted administrator access' : 'Revoked administrator access',
          'OK',
          { duration: snackSuccessDuration },
        );
        this.changed.emit();
      },
      error: () => {
        this.loading.set(false);
        this.checked.set(previous);
        this.snackBar.open('Failed to update administrator access', 'OK', {
          duration: snackErrorDuration,
        });
      },
    });
  }
}
