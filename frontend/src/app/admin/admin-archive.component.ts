import {
  ChangeDetectionStrategy,
  Component,
  effect,
  inject,
  input,
  output,
  signal,
} from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatIconModule } from '@angular/material/icon';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { Api } from '~/api/api';
import { updateUserAdmin } from '~/api/functions';
import { snackErrorDuration, snackSuccessDuration } from '~/frontend/constants';

@Component({
  selector: 'app-admin-archive',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [MatButtonModule, MatCardModule, MatIconModule, MatSnackBarModule],
  templateUrl: './admin-archive.component.html',
})
export class AdminArchiveComponent {
  private readonly api = inject(Api);
  private readonly snackBar = inject(MatSnackBar);

  readonly userId = input.required<string>();
  readonly archived = input.required<boolean>();
  readonly changed = output<void>();

  readonly currentlyArchived = signal(false);
  readonly loading = signal(false);

  constructor() {
    effect(() => {
      this.currentlyArchived.set(this.archived());
    });
  }

  toggle() {
    if (this.currentlyArchived()) {
      this.runUnarchive();
    } else {
      this.runArchive();
    }
  }

  private async runArchive() {
    this.loading.set(true);
    try {
      await this.api.invoke(updateUserAdmin, { id: this.userId(), body: { archived: true } });
      this.currentlyArchived.set(true);
      const ref = this.snackBar.open('User archived', 'Undo', { duration: snackSuccessDuration });
      ref.onAction().subscribe(() => this.runUnarchive());
      this.changed.emit();
    } catch {
      this.snackBar.open('Failed to archive user', 'OK', { duration: snackErrorDuration });
    } finally {
      this.loading.set(false);
    }
  }

  private async runUnarchive() {
    this.loading.set(true);
    try {
      await this.api.invoke(updateUserAdmin, { id: this.userId(), body: { archived: false } });
      this.currentlyArchived.set(false);
      this.snackBar.open('User unarchived', 'OK', { duration: snackSuccessDuration });
      this.changed.emit();
    } catch {
      this.snackBar.open('Failed to unarchive user', 'OK', { duration: snackErrorDuration });
    } finally {
      this.loading.set(false);
    }
  }
}
