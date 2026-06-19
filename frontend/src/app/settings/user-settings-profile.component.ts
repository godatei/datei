import { Component, inject, signal } from '@angular/core';
import { MatDividerModule } from '@angular/material/divider';
import { AuthService } from '~/frontend/services/auth.service';
import { SettingsService } from '~/frontend/services/settings.service';
import { createSelfUserPort, UserSnapshot } from '~/frontend/users/user-data.port';
import { UserEmailsComponent } from '~/frontend/users/user-emails/user-emails.component';
import { UserProfileComponent } from '~/frontend/users/user-profile/user-profile.component';

@Component({
  selector: 'app-user-settings-profile',
  imports: [MatDividerModule, UserProfileComponent, UserEmailsComponent],
  templateUrl: './user-settings-profile.component.html',
})
export class UserSettingsProfileComponent {
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
