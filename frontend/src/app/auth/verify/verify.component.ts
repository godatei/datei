import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { Router } from '@angular/router';
import { AuthService } from '~/frontend/services/auth.service';
import { SettingsService } from '~/frontend/services/settings.service';

@Component({
  selector: 'app-verify',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [MatCardModule, MatButtonModule, MatIconModule, MatProgressSpinnerModule],
  templateUrl: './verify.component.html',
  host: { class: 'block' },
})
export class VerifyComponent {
  private readonly settings = inject(SettingsService);
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);

  readonly loading = signal(false);

  resend() {
    this.loading.set(true);
    this.settings.requestEmailVerification().subscribe({
      next: () => this.loading.set(false),
      error: () => this.loading.set(false),
    });
  }

  logout() {
    this.auth.logout();
    this.router.navigate(['/login']);
  }
}
