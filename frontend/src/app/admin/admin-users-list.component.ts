import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  OnInit,
  signal,
} from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatChipsModule } from '@angular/material/chips';
import { MatIconModule } from '@angular/material/icon';
import { MatRippleModule } from '@angular/material/core';
import { MatDialog } from '@angular/material/dialog';
import { MatTooltipModule } from '@angular/material/tooltip';
import { RouterLink } from '@angular/router';
import type { AdminUserListItem } from '~/api/models/admin-user-list-item';
import { AdminUsersService } from '~/frontend/services/admin-users.service';
import { initials } from '~/frontend/users/initials';
import { AdminCreateUserDialogComponent } from './admin-create-user-dialog.component';

@Component({
  selector: 'app-admin-users-list',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatButtonModule,
    MatCardModule,
    MatChipsModule,
    MatIconModule,
    MatRippleModule,
    MatTooltipModule,
    RouterLink,
  ],
  templateUrl: './admin-users-list.component.html',
})
export class AdminUsersListComponent implements OnInit {
  private readonly admin = inject(AdminUsersService);
  private readonly dialog = inject(MatDialog);

  readonly users = signal<AdminUserListItem[]>([]);
  readonly loading = signal(true);
  readonly error = signal(false);
  readonly filter = signal<'all' | 'admins' | 'users' | 'archived'>('all');

  readonly visibleUsers = computed(() => {
    const f = this.filter();
    const all = this.users();
    if (f === 'admins') return all.filter((u) => u.isAdmin && !u.archived);
    if (f === 'users') return all.filter((u) => !u.isAdmin && !u.archived);
    if (f === 'archived') return all.filter((u) => u.archived);
    return all;
  });

  ngOnInit() {
    this.load();
  }

  private load() {
    this.loading.set(true);
    this.admin.listUsers().subscribe({
      next: (users) => {
        this.users.set(users);
        this.loading.set(false);
      },
      error: () => {
        this.loading.set(false);
        this.error.set(true);
      },
    });
  }

  openCreate() {
    const ref = this.dialog.open<
      AdminCreateUserDialogComponent,
      undefined,
      AdminUserListItem | undefined
    >(AdminCreateUserDialogComponent, { width: '480px' });
    ref.afterClosed().subscribe((created) => {
      if (created) this.load();
    });
  }

  initials(name: string) {
    return initials(name);
  }
}
