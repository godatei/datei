import { DatePipe } from '@angular/common';
import { HttpErrorResponse } from '@angular/common/http';
import { Component, computed, inject, resource, signal } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
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
import { UserPatCreateDialogComponent } from '../user-pat-create-dialog/user-pat-create-dialog.component';
import { UserPatRevokeDialogComponent } from '../user-pat-revoke-dialog/user-pat-revoke-dialog.component';

@Component({
  selector: 'app-user-personal-access-tokens',
  imports: [
    DatePipe,
    MatButtonModule,
    MatDividerModule,
    MatIconModule,
    MatSnackBarModule,
    MatListModule,
  ],
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
      await this.revokeWithRetry(token.id);
      this.reloadTokens();
      this.snackBar.open('Access token revoked', 'OK', { duration: snackSuccessDuration });
    } catch (e) {
      console.error(e);
      // Reconcile the list with the server: a concurrent write may have changed
      // it (e.g. the token was already revoked elsewhere) even though we failed.
      this.reloadTokens();
      this.snackBar.open('Failed to revoke access token', 'Dismiss', {
        duration: snackErrorDuration,
      });
    } finally {
      this.revokingTokenId.set(null);
    }
  }

  // The API returns 409 for optimistic-lock conflicts on the shared user stream
  // and asks the client to retry; the revoke did not take effect in that case.
  private async revokeWithRetry(id: string, attempts = 3): Promise<void> {
    for (let attempt = 0; attempt < attempts; attempt++) {
      try {
        await this.api.invoke(revokePersonalAccessToken, { id });
        return;
      } catch (e) {
        if (e instanceof HttpErrorResponse && e.status === 409 && attempt < attempts - 1) {
          continue;
        }
        throw e;
      }
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
