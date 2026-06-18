import { Component, computed, inject, resource, signal } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatChipsModule } from '@angular/material/chips';
import { MatDividerModule } from '@angular/material/divider';
import { MatIconModule } from '@angular/material/icon';
import { MatListModule } from '@angular/material/list';
import { MatDialog } from '@angular/material/dialog';
import { MatTooltipModule } from '@angular/material/tooltip';
import { RouterLink } from '@angular/router';
import { Api } from '~/api/api';
import { listUsersAdmin } from '~/api/functions';
import type { AdminUserListItem } from '~/api/models/admin-user-list-item';
import { UserAvatarComponent } from '~/frontend/users/user-avatar.component';
import { AdminCreateUserDialogComponent } from './admin-create-user-dialog.component';

@Component({
  selector: 'app-admin-users-list',
  imports: [
    MatButtonModule,
    MatCardModule,
    MatChipsModule,
    MatDividerModule,
    MatIconModule,
    MatListModule,
    MatTooltipModule,
    RouterLink,
    UserAvatarComponent,
  ],
  templateUrl: './admin-users-list.component.html',
  styleUrl: './admin-users-list.component.css',
})
export class AdminUsersListComponent {
  private readonly api = inject(Api);
  private readonly dialog = inject(MatDialog);

  readonly filter = signal<'all' | 'admins' | 'users' | 'archived'>('all');

  protected readonly usersResource = resource({
    loader: () => this.api.invoke(listUsersAdmin),
  });

  readonly users = computed<AdminUserListItem[]>(() => this.usersResource.value()?.items ?? []);
  readonly loading = computed(() => this.usersResource.isLoading());
  readonly error = computed(() => this.usersResource.error() !== undefined);

  readonly visibleUsers = computed(() => {
    const f = this.filter();
    const all = this.users();
    if (f === 'admins') return all.filter((u) => u.isAdmin && !u.archived);
    if (f === 'users') return all.filter((u) => !u.isAdmin && !u.archived);
    if (f === 'archived') return all.filter((u) => u.archived);
    return all;
  });

  openCreate() {
    const ref = this.dialog.open<
      AdminCreateUserDialogComponent,
      undefined,
      AdminUserListItem | undefined
    >(AdminCreateUserDialogComponent, { width: '480px' });
    ref.afterClosed().subscribe((created) => {
      if (created) this.usersResource.reload();
    });
  }
}
