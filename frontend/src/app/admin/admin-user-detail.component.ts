import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  OnInit,
  signal,
} from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatIconModule } from '@angular/material/icon';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { AdminUsersService } from '~/frontend/services/admin-users.service';
import { AuthService } from '~/frontend/services/auth.service';
import { UserAvatarComponent } from '~/frontend/users/user-avatar.component';
import { createAdminUserPort, UserSnapshot } from '~/frontend/users/user-data.port';
import { UserEmailsComponent } from '~/frontend/users/user-emails.component';
import { UserPasswordComponent } from '~/frontend/users/user-password.component';
import { UserProfileComponent } from '~/frontend/users/user-profile.component';
import { AdminArchiveComponent } from './admin-archive.component';
import { AdminMfaComponent } from './admin-mfa.component';
import { AdminRoleComponent } from './admin-role.component';

@Component({
  selector: 'app-admin-user-detail',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatButtonModule,
    MatCardModule,
    MatIconModule,
    RouterLink,
    UserAvatarComponent,
    UserProfileComponent,
    UserEmailsComponent,
    UserPasswordComponent,
    AdminMfaComponent,
    AdminRoleComponent,
    AdminArchiveComponent,
  ],
  templateUrl: './admin-user-detail.component.html',
})
export class AdminUserDetailComponent implements OnInit {
  private readonly admin = inject(AdminUsersService);
  private readonly auth = inject(AuthService);
  private readonly route = inject(ActivatedRoute);

  private readonly params = toSignal(this.route.paramMap);
  readonly userId = computed(() => this.params()?.get('id') ?? '');
  readonly isSelf = computed(() => this.auth.getClaims()?.sub === this.userId());

  readonly user = signal<UserSnapshot | undefined>(undefined);
  readonly primaryEmail = signal<string | null>(null);
  readonly loading = signal(true);

  readonly port = computed(() => createAdminUserPort(this.admin, this.userId()));

  ngOnInit() {
    this.load();
  }

  load() {
    const id = this.userId();
    if (!id) {
      this.loading.set(false);
      return;
    }
    this.admin.getUser(id).subscribe({
      next: (u) => {
        this.user.set({
          name: u.name,
          isAdmin: u.isAdmin,
          mfaEnabled: u.mfaEnabled,
          archived: u.archived,
        });
        this.primaryEmail.set(u.primaryEmail ?? null);
        this.loading.set(false);
      },
      error: () => this.loading.set(false),
    });
  }
}
