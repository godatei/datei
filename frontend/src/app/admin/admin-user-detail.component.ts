import { Component, computed, inject, resource } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatIconModule } from '@angular/material/icon';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { Api } from '~/api/api';
import { getUserAdmin } from '~/api/functions';
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
export class AdminUserDetailComponent {
  private readonly api = inject(Api);
  private readonly auth = inject(AuthService);
  private readonly route = inject(ActivatedRoute);

  private readonly params = toSignal(this.route.paramMap);
  readonly userId = computed(() => this.params()?.get('id') ?? '');
  readonly isSelf = computed(() => this.auth.getClaims()?.sub === this.userId());

  protected readonly userResource = resource({
    params: () => ({ id: this.userId() }),
    loader: async ({ params }) => {
      if (!params.id) return undefined;
      return this.api.invoke(getUserAdmin, { id: params.id });
    },
  });

  readonly user = computed<UserSnapshot | undefined>(() => {
    const u = this.userResource.value();
    if (!u) return undefined;
    return { name: u.name, isAdmin: u.isAdmin, mfaEnabled: u.mfaEnabled, archived: u.archived };
  });
  readonly primaryEmail = computed(() => this.userResource.value()?.primaryEmail ?? null);
  readonly loading = computed(() => this.userResource.isLoading());

  readonly port = computed(() => createAdminUserPort(this.api, this.userId()));

  load() {
    this.userResource.reload();
  }
}
