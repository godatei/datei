import { DatePipe } from '@angular/common';
import { HttpErrorResponse } from '@angular/common/http';
import { Component, computed, inject, resource, signal } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatDialog } from '@angular/material/dialog';
import { MatIconModule } from '@angular/material/icon';
import { MatListModule } from '@angular/material/list';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { isPast } from 'date-fns';
import { firstValueFrom } from 'rxjs';
import { Api } from '~/api/api';
import { listPersonalAccessTokens, revokePersonalAccessToken } from '~/api/functions';
import type { PersonalAccessToken } from '~/api/models/personal-access-token';
import { snackErrorDuration, snackSuccessDuration } from '~/frontend/constants';
import { retryOnConflict } from '~/util/retry-on-conflict';
import { UserPatCreateDialogComponent } from '../user-pat-create-dialog/user-pat-create-dialog.component';
import { UserPatRevokeDialogComponent } from '../user-pat-revoke-dialog/user-pat-revoke-dialog.component';

@Component({
  selector: 'app-user-personal-access-tokens',
  imports: [DatePipe, MatButtonModule, MatIconModule, MatSnackBarModule, MatListModule],
  templateUrl: './user-personal-access-tokens.component.html',
})
export class UserPersonalAccessTokensComponent {
  private readonly api = inject(Api);
  private readonly dialog = inject(MatDialog);
  private readonly snackBar = inject(MatSnackBar);

  private readonly reloadKey = signal(0);
  protected readonly revokingTokenId = signal<string | null>(null);

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
    this.reloadTokens();
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
      await retryOnConflict(() => this.api.invoke(revokePersonalAccessToken, { id: token.id }));
      this.reloadTokens();
      this.snackBar.open('Access token revoked', 'OK', { duration: snackSuccessDuration });
    } catch (e) {
      // 404 means the token was already revoked (double-click, two tabs, or a 409
      // retry that raced a concurrent revoke). The desired end state is reached,
      // so treat it as success rather than surfacing a misleading error.
      if (e instanceof HttpErrorResponse && e.status === 404) {
        this.reloadTokens();
        this.snackBar.open('Access token revoked', 'OK', { duration: snackSuccessDuration });
        return;
      }
      console.error(e);
      // Reconcile the list with the server: a concurrent write may have changed
      // it even though we failed.
      this.reloadTokens();
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
