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
import { Api } from '~/api/api';
import { updateUserAdmin } from '~/api/functions';
import { snackErrorDuration, snackSuccessDuration } from '~/frontend/constants';

@Component({
  selector: 'app-admin-role',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [MatCardModule, MatSlideToggleModule, MatSnackBarModule],
  templateUrl: './admin-role.component.html',
})
export class AdminRoleComponent {
  private readonly api = inject(Api);
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

  async toggle(value: boolean) {
    const previous = this.checked();
    this.checked.set(value);
    this.loading.set(true);
    try {
      await this.api.invoke(updateUserAdmin, { id: this.userId(), body: { isAdmin: value } });
      this.snackBar.open(
        value ? 'Granted administrator access' : 'Revoked administrator access',
        'OK',
        { duration: snackSuccessDuration },
      );
      this.changed.emit();
    } catch {
      this.checked.set(previous);
      this.snackBar.open('Failed to update administrator access', 'OK', {
        duration: snackErrorDuration,
      });
    } finally {
      this.loading.set(false);
    }
  }
}
