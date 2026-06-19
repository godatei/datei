import { Component, inject, signal } from '@angular/core';
import { AuthService } from '~/frontend/services/auth.service';
import { SettingsService } from '~/frontend/services/settings.service';
import { createSelfUserPort, UserSnapshot } from '~/frontend/users/user-data.port';
import { UserMfaComponent } from '~/frontend/users/user-mfa.component';
import { UserPasswordComponent } from '~/frontend/users/user-password.component';
import { UserPersonalAccessTokensComponent } from '~/frontend/users/user-personal-access-tokens.component';

@Component({
  selector: 'app-user-settings-security',
  imports: [UserPasswordComponent, UserMfaComponent, UserPersonalAccessTokensComponent],
  templateUrl: './user-settings-security.component.html',
})
export class UserSettingsSecurityComponent {
  private readonly settings = inject(SettingsService);
  private readonly auth = inject(AuthService);

  protected readonly port = createSelfUserPort(this.settings, this.auth);
  protected readonly user = signal<UserSnapshot | undefined>(undefined);

  constructor() {
    this.load();
  }

  protected load() {
    this.port.load().subscribe({
      next: (u) => this.user.set(u),
    });
  }
}
