import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { AuthService } from '~/frontend/services/auth.service';
import { SettingsService } from '~/frontend/services/settings.service';
import { createSelfUserPort, UserSnapshot } from '~/frontend/users/user-data.port';
import { UserEmailsComponent } from '~/frontend/users/user-emails.component';
import { UserMfaComponent } from '~/frontend/users/user-mfa.component';
import { UserPasswordComponent } from '~/frontend/users/user-password.component';
import { UserProfileComponent } from '~/frontend/users/user-profile.component';

@Component({
  selector: 'app-user-settings',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [UserProfileComponent, UserEmailsComponent, UserPasswordComponent, UserMfaComponent],
  templateUrl: './user-settings.component.html',
})
export class UserSettingsComponent {
  private readonly settings = inject(SettingsService);
  private readonly auth = inject(AuthService);

  readonly port = createSelfUserPort(this.settings, this.auth);

  readonly user = signal<UserSnapshot | undefined>(undefined);
  readonly name = computed(() => this.user()?.name ?? '');
  readonly mfaEnabled = computed(() => this.user()?.mfaEnabled ?? false);

  constructor() {
    this.load();
  }

  load() {
    this.port.load().subscribe({
      next: (u) => this.user.set(u),
    });
  }
}
