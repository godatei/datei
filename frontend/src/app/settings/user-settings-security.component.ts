import { Component, inject, signal } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatDividerModule } from '@angular/material/divider';
import { AuthService } from '~/frontend/services/auth.service';
import { SettingsService } from '~/frontend/services/settings.service';
import { createSelfUserPort, UserSnapshot } from '~/frontend/users/user-data.port';
import { UserMfaComponent } from '~/frontend/users/user-mfa/user-mfa.component';
import { UserPasswordComponent } from '~/frontend/users/user-password/user-password.component';
import { UserPersonalAccessTokensComponent } from '~/frontend/users/user-personal-access-tokens/user-personal-access-tokens.component';

@Component({
  selector: 'app-user-settings-security',
  imports: [
    MatButtonModule,
    MatDividerModule,
    UserPasswordComponent,
    UserMfaComponent,
    UserPersonalAccessTokensComponent,
  ],
  templateUrl: './user-settings-security.component.html',
})
export class UserSettingsSecurityComponent {
  private readonly settings = inject(SettingsService);
  private readonly auth = inject(AuthService);

  protected readonly port = createSelfUserPort(this.settings, this.auth);
  protected readonly user = signal<UserSnapshot | undefined>(undefined);
  protected readonly loadFailed = signal(false);

  constructor() {
    this.load();
  }

  protected load() {
    this.loadFailed.set(false);
    this.port.load().subscribe({
      next: (u) => this.user.set(u),
      error: () => this.loadFailed.set(true),
    });
  }

  // Silent refresh after an MFA change: updates mfaEnabled without tearing down
  // the view (which would discard the recovery codes shown by the MFA child).
  protected refresh() {
    this.port.load().subscribe({
      next: (u) => this.user.set(u),
    });
  }
}
