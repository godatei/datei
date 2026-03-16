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
  template: `
    <div class="auth-container">
      <mat-card class="auth-card">
        <mat-card-content>
          <div class="auth-header">
            <mat-icon class="brand-icon">mark_email_unread</mat-icon>
            <h1>Verify your email</h1>
            <p class="subtitle">Check your inbox for a verification link</p>
          </div>

          <p class="success-message">
            We sent a verification email to your address. Click the link in the email to verify your
            account.
          </p>

          <div class="verify-actions">
            <button class="submit-btn" mat-flat-button (click)="resend()" [disabled]="loading()">
              @if (loading()) {
                <mat-spinner diameter="20"></mat-spinner>
              } @else {
                Resend verification email
              }
            </button>

            <button class="submit-btn" mat-button (click)="logout()">Logout</button>
          </div>
        </mat-card-content>
      </mat-card>
    </div>
  `,
  styleUrls: ['../auth-shared.css'],
  styles: `
    .verify-actions {
      display: flex;
      flex-direction: column;
      gap: 0.5rem;
    }
  `,
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
