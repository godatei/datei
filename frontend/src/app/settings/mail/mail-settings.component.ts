import { Component, computed, inject, resource, signal } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatDialog } from '@angular/material/dialog';
import { MatIconModule } from '@angular/material/icon';
import { MatListModule } from '@angular/material/list';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import {
  MAT_TOOLTIP_DEFAULT_OPTIONS,
  MatTooltipDefaultOptions,
  MatTooltipModule,
} from '@angular/material/tooltip';
import { firstValueFrom } from 'rxjs';
import { Api } from '~/api/api';
import {
  deleteMailAccount,
  deleteMailRule,
  listAllMailRules,
  listMailAccounts,
  testMailAccount,
  updateMailRule,
} from '~/api/functions';
import type { MailAccount } from '~/api/models/mail-account';
import type { MailRule } from '~/api/models/mail-rule';
import { snackErrorDuration, snackSuccessDuration } from '~/frontend/constants';
import { retryOnConflict } from '~/util/retry-on-conflict';
import {
  ConfirmDialogComponent,
  ConfirmDialogData,
} from '~/frontend/components/confirm-dialog.component';
import { MailAccountDialogComponent, MailAccountDialogData } from './mail-account-dialog.component';
import { MailRuleDialogComponent, MailRuleDialogData } from './mail-rule-dialog.component';

@Component({
  selector: 'app-mail-settings',
  templateUrl: './mail-settings.component.html',
  imports: [
    MatButtonModule,
    MatIconModule,
    MatListModule,
    MatSlideToggleModule,
    MatSnackBarModule,
    MatTooltipModule,
  ],
  providers: [
    {
      provide: MAT_TOOLTIP_DEFAULT_OPTIONS,
      useValue: {
        showDelay: 500,
        hideDelay: 50,
        touchendHideDelay: 50,
      } satisfies MatTooltipDefaultOptions,
    },
  ],
})
export class MailSettingsComponent {
  private readonly api = inject(Api);
  private readonly dialog = inject(MatDialog);
  private readonly snackBar = inject(MatSnackBar);

  private readonly reloadKey = signal(0);
  protected readonly testingAccountId = signal<string | null>(null);

  protected readonly accountsResource = resource({
    params: () => this.reloadKey(),
    loader: async () => (await this.api.invoke(listMailAccounts, { limit: 200 })).items,
  });

  protected readonly rulesResource = resource({
    params: () => this.reloadKey(),
    loader: async () => (await this.api.invoke(listAllMailRules, { limit: 500 })).items,
  });

  protected readonly accounts = computed(() => this.accountsResource.value() ?? []);
  protected readonly rules = computed(() => this.rulesResource.value() ?? []);

  protected readonly loading = computed(
    () => this.accountsResource.isLoading() || this.rulesResource.isLoading(),
  );
  protected readonly loadingError = computed(
    () => this.accountsResource.error() ?? this.rulesResource.error(),
  );

  private readonly accountNames = computed(() => {
    const map = new Map<string, string>();
    for (const account of this.accounts()) map.set(account.id, account.name);
    return map;
  });

  protected accountName(accountId: string): string {
    return this.accountNames().get(accountId) ?? 'Unknown account';
  }

  protected async addAccount(): Promise<void> {
    const ref = this.dialog.open<MailAccountDialogComponent, MailAccountDialogData, MailAccount>(
      MailAccountDialogComponent,
    );
    const created = await firstValueFrom(ref.afterClosed());
    if (created) this.reload();
  }

  protected async editAccount(account: MailAccount): Promise<void> {
    const ref = this.dialog.open<MailAccountDialogComponent, MailAccountDialogData, MailAccount>(
      MailAccountDialogComponent,
      { data: { account } },
    );
    const updated = await firstValueFrom(ref.afterClosed());
    if (updated) this.reload();
  }

  protected async deleteAccount(account: MailAccount): Promise<void> {
    const confirmed = await this.confirm({
      title: 'Delete mail account',
      message: `Delete "${account.name}" and all of its rules? This cannot be undone.`,
      confirmLabel: 'Delete',
    });
    if (!confirmed) return;

    try {
      await retryOnConflict(() => this.api.invoke(deleteMailAccount, { accountId: account.id }));
      this.snackBar.open('Mail account deleted', 'OK', { duration: snackSuccessDuration });
      this.reload();
    } catch (e) {
      console.error(e);
      this.snackBar.open('Failed to delete mail account', 'Dismiss', {
        duration: snackErrorDuration,
      });
    }
  }

  protected async testAccount(account: MailAccount): Promise<void> {
    this.testingAccountId.set(account.id);
    try {
      const result = await this.api.invoke(testMailAccount, { accountId: account.id });
      if (result.success) {
        this.snackBar.open('Connection successful', 'OK', { duration: snackSuccessDuration });
      } else {
        this.snackBar.open(`Connection failed: ${result.message ?? 'unknown error'}`, 'Dismiss', {
          duration: snackErrorDuration,
        });
      }
    } catch (e) {
      console.error(e);
      this.snackBar.open('Failed to test connection', 'Dismiss', { duration: snackErrorDuration });
    } finally {
      this.testingAccountId.set(null);
    }
  }

  protected async addRule(): Promise<void> {
    if (this.accounts().length === 0) {
      this.snackBar.open('Add a mail account before creating rules', 'OK', {
        duration: snackErrorDuration,
      });
      return;
    }
    const ref = this.dialog.open<MailRuleDialogComponent, MailRuleDialogData, MailRule>(
      MailRuleDialogComponent,
      { width: '46rem', maxWidth: '95vw', data: { accounts: this.accounts() } },
    );
    const created = await firstValueFrom(ref.afterClosed());
    if (created) this.reload();
  }

  protected async editRule(rule: MailRule): Promise<void> {
    const ref = this.dialog.open<MailRuleDialogComponent, MailRuleDialogData, MailRule>(
      MailRuleDialogComponent,
      { width: '46rem', maxWidth: '95vw', data: { accounts: this.accounts(), rule } },
    );
    const updated = await firstValueFrom(ref.afterClosed());
    if (updated) this.reload();
  }

  protected async toggleRule(rule: MailRule, enabled: boolean): Promise<void> {
    const body = { ...rule, enabled };

    try {
      await retryOnConflict(() => this.api.invoke(updateMailRule, { ruleId: rule.id, body }));
      this.reload();
    } catch (e) {
      console.error(e);
      this.snackBar.open('Failed to update rule', 'Dismiss', { duration: snackErrorDuration });
      this.reload();
    }
  }

  protected async deleteRule(rule: MailRule): Promise<void> {
    const confirmed = await this.confirm({
      title: 'Delete rule',
      message: `Delete rule "${rule.name}"? This cannot be undone.`,
      confirmLabel: 'Delete',
    });
    if (!confirmed) return;

    try {
      await retryOnConflict(() => this.api.invoke(deleteMailRule, { ruleId: rule.id }));
      this.snackBar.open('Rule deleted', 'OK', { duration: snackSuccessDuration });
      this.reload();
    } catch (e) {
      console.error(e);
      this.snackBar.open('Failed to delete rule', 'Dismiss', { duration: snackErrorDuration });
    }
  }

  private async confirm(data: ConfirmDialogData): Promise<boolean> {
    const ref = this.dialog.open<ConfirmDialogComponent, ConfirmDialogData, boolean>(
      ConfirmDialogComponent,
      { data },
    );
    return (await firstValueFrom(ref.afterClosed())) ?? false;
  }

  private reload(): void {
    this.reloadKey.update((v) => v + 1);
  }
}
