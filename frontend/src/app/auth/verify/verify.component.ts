import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { Router } from '@angular/router';
import { AuthService } from '~/frontend/services/auth.service';
import { SettingsService } from '~/frontend/services/settings.service';

@Component({
  selector: 'app-verify',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [MatCardModule, MatButtonModule, MatProgressSpinnerModule],
  template: `
    <div class="auth-container">
      <mat-card>
        <mat-card-header>
          <mat-card-title>Verify your email</mat-card-title>
        </mat-card-header>
        <mat-card-content>
          <p>
            Please check your inbox for a verification email and click the link to verify your
            address.
          </p>

          <button mat-flat-button (click)="resend()" [disabled]="loading()">
            @if (loading()) {
              <mat-spinner diameter="20"></mat-spinner>
            } @else {
              Resend verification email
            }
          </button>

          <button mat-button (click)="logout()">Logout</button>
        </mat-card-content>
      </mat-card>
    </div>
  `,
  styles: `
    .auth-container {
      display: flex;
      justify-content: center;
      align-items: center;
      min-height: 100vh;
      padding: 1rem;
    }
    mat-card {
      max-width: 400px;
      width: 100%;
    }
    mat-card-content {
      display: flex;
      flex-direction: column;
      gap: 1rem;
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
