import { Component, computed, inject, signal } from '@angular/core';
import { form, FormField, FormRoot, min, required } from '@angular/forms/signals';
import { MatButtonModule } from '@angular/material/button';
import {
  MAT_DIALOG_DATA,
  MatDialog,
  MatDialogModule,
  MatDialogRef,
} from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { firstValueFrom } from 'rxjs';
import { Api } from '~/api/api';
import { createMailRule, updateMailRule } from '~/api/functions';
import type { MailAccount } from '~/api/models/mail-account';
import type { MailRule } from '~/api/models/mail-rule';
import { retryOnConflict } from '~/util/retry-on-conflict';
import {
  DirectoryPickerDialogComponent,
  DirectorySelection,
} from './directory-picker-dialog.component';

export interface MailRuleDialogData {
  accounts: MailAccount[];
  rule?: MailRule;
}

interface MailRuleFormModel {
  accountId: string;
  name: string;
  order: number;
  enabled: boolean;
  folder: string;
  filterFrom: string;
  filterSubject: string;
  maxAgeDays: number;
  attachmentPattern: string;
}

@Component({
  selector: 'app-mail-rule-dialog',
  templateUrl: './mail-rule-dialog.component.html',
  imports: [
    FormField,
    FormRoot,
    MatButtonModule,
    MatDialogModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
  ],
})
export class MailRuleDialogComponent {
  private readonly api = inject(Api);
  private readonly dialog = inject(MatDialog);
  private readonly data = inject<MailRuleDialogData>(MAT_DIALOG_DATA);
  private readonly dialogRef = inject(MatDialogRef<MailRuleDialogComponent, MailRule | undefined>);

  protected readonly isEdit = computed(() => this.data.rule !== undefined);
  protected readonly accounts = this.data.accounts;
  protected readonly errorMessage = signal<string | null>(null);

  protected readonly targetDirectoryId = signal<string | undefined>(
    this.data.rule?.targetDirectoryId,
  );
  protected readonly targetDirectoryLabel = signal<string>(
    this.data.rule?.targetDirectoryId ? 'Selected directory' : 'Root',
  );

  protected readonly model = signal<MailRuleFormModel>({
    accountId: this.data.rule?.accountId ?? this.data.accounts[0]?.id ?? '',
    name: this.data.rule?.name ?? '',
    order: this.data.rule?.order ?? 0,
    enabled: this.data.rule?.enabled ?? true,
    folder: this.data.rule?.folder ?? 'INBOX',
    filterFrom: this.data.rule?.filterFrom ?? '',
    filterSubject: this.data.rule?.filterSubject ?? '',
    maxAgeDays: this.data.rule?.maxAgeDays ?? 30,
    attachmentPattern: this.data.rule?.attachmentPattern ?? '',
  });

  protected readonly ruleForm = form(
    this.model,
    (p) => {
      required(p.accountId);
      required(p.name);
      required(p.folder);
      min(p.maxAgeDays, 1);
      min(p.order, 0);
    },
    {
      submission: {
        action: async () => {
          this.errorMessage.set(null);
          const m = this.model();

          const body = {
            accountId: m.accountId,
            name: m.name.trim(),
            order: m.order,
            enabled: m.enabled,
            folder: m.folder.trim(),
            maxAgeDays: m.maxAgeDays,
            action: 'mark_as_read' as const,
            ...optional('filterFrom', m.filterFrom),
            ...optional('filterSubject', m.filterSubject),
            ...optional('attachmentPattern', m.attachmentPattern),
            ...(this.targetDirectoryId() ? { targetDirectoryId: this.targetDirectoryId() } : {}),
          };

          try {
            const rule = this.data.rule;
            if (rule) {
              const result = await retryOnConflict(() =>
                this.api.invoke(updateMailRule, { ruleId: rule.id, body }),
              );
              this.dialogRef.close(result);
            } else {
              const result = await retryOnConflict(() => this.api.invoke(createMailRule, { body }));
              this.dialogRef.close(result);
            }
          } catch (e) {
            console.error(e);
            this.errorMessage.set('Failed to save rule');
          }
        },
      },
    },
  );

  protected async chooseDirectory(): Promise<void> {
    const dialogRef = this.dialog.open<
      DirectoryPickerDialogComponent,
      unknown,
      DirectorySelection | undefined
    >(DirectoryPickerDialogComponent);
    const selection = await firstValueFrom(dialogRef.afterClosed());
    if (!selection) return;
    this.targetDirectoryId.set(selection.id);
    this.targetDirectoryLabel.set(selection.name);
  }

  protected cancel(): void {
    this.dialogRef.close(undefined);
  }
}

function optional(key: string, value: string): Record<string, string> {
  const trimmed = value.trim();
  return trimmed === '' ? {} : { [key]: trimmed };
}
