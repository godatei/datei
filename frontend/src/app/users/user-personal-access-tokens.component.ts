import { DatePipe } from '@angular/common';
import { Component, computed, inject, resource, signal } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatDialog } from '@angular/material/dialog';
import { MatDividerModule } from '@angular/material/divider';
import { MatIconModule } from '@angular/material/icon';
import { MatListModule } from '@angular/material/list';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { isPast } from 'date-fns';
import { firstValueFrom } from 'rxjs';
import { Api } from '~/api/api';
import { listPersonalAccessTokens, revokePersonalAccessToken } from '~/api/functions';
import type { PersonalAccessToken } from '~/api/models/personal-access-token';
import { snackErrorDuration, snackSuccessDuration } from '~/frontend/constants';
import { UserPatCreateDialogComponent } from './user-pat-create-dialog.component';
import { UserPatRevokeDialogComponent } from './user-pat-revoke-dialog.component';

@Component({
  selector: 'app-user-personal-access-tokens',
  imports: [
    DatePipe,
    MatButtonModule,
    MatCardModule,
    MatDividerModule,
    MatIconModule,
    MatSnackBarModule,
    MatListModule,
  ],
  templateUrl: './user-personal-access-tokens.component.html',
  styles: [
    `
      @reference 'tailwindcss';

      .pat-item-meta {
        @apply h-full flex! items-center;
      }
    `,
  ],
})
export class UserPersonalAccessTokensComponent {
  private readonly api = inject(Api);
  private readonly dialog = inject(MatDialog);
  private readonly snackBar = inject(MatSnackBar);

  private readonly reloadKey = signal(0);
  protected readonly revokingTokenId = signal<string | null>(null);
  protected readonly createdToken = signal<string | null>(null);

  protected readonly tokensResource = resource({
    params: () => this.reloadKey(),
    loader: async () => (await this.api.invoke(listPersonalAccessTokens, undefined)).tokens,
  });

  protected readonly tokens = computed(() => this.tokensResource.value() ?? []);
  protected readonly loading = computed(() => this.tokensResource.isLoading());
  protected readonly loadingError = computed(() => this.tokensResource.error());

  protected async openCreateDialog(): Promise<void> {
    const dialogRef = this.dialog.open(UserPatCreateDialogComponent);
    const created = await firstValueFrom(dialogRef.afterClosed());
    if (!created) return;

    this.createdToken.set(created.token);
    this.reloadTokens();
    this.snackBar.open('Access token created', 'OK', { duration: snackSuccessDuration });
  }

  protected clearCreatedToken(): void {
    this.createdToken.set(null);
  }

  protected isRevoking(token: PersonalAccessToken): boolean {
    return this.revokingTokenId() === token.id;
  }

  protected async revokeToken(token: PersonalAccessToken): Promise<void> {
    const dialogRef = this.dialog.open(UserPatRevokeDialogComponent, {
      data: { label: this.displayLabel(token) },
    });
    const confirmed = await firstValueFrom(dialogRef.afterClosed());
    if (!confirmed) return;

    this.revokingTokenId.set(token.id);
    try {
      await this.api.invoke(revokePersonalAccessToken, { id: token.id });
      this.reloadTokens();
      this.snackBar.open('Access token revoked', 'OK', { duration: snackSuccessDuration });
    } catch (e) {
      console.error(e);
      this.snackBar.open('Failed to revoke access token', 'Dismiss', {
        duration: snackErrorDuration,
      });
    } finally {
      this.revokingTokenId.set(null);
    }
  }

  protected displayLabel(token: PersonalAccessToken): string {
    const label = token.label?.trim();
    return label ? label : 'No label';
  }

  protected expired(token: PersonalAccessToken): boolean {
    return token.expiresAt !== undefined && isPast(new Date(token.expiresAt));
  }

  private reloadTokens(): void {
    this.reloadKey.update((value) => value + 1);
  }
}
