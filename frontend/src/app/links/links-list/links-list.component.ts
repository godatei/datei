import { DatePipe } from '@angular/common';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  inject,
  resource,
  signal,
} from '@angular/core';
import { Clipboard } from '@angular/cdk/clipboard';
import { MatButtonModule } from '@angular/material/button';
import { MatChipsModule } from '@angular/material/chips';
import { MatDialog } from '@angular/material/dialog';
import { MatIconModule } from '@angular/material/icon';
import { MatMenuModule } from '@angular/material/menu';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatTableDataSource, MatTableModule } from '@angular/material/table';
import { MatTooltipModule } from '@angular/material/tooltip';
import { Api } from '~/api/api';
import { getLink, listLinks, revokeLink, rotateLinkKey } from '~/api/functions';
import type { Link } from '~/api/models/link';
import { snackErrorDuration, snackSuccessDuration } from '~/frontend/constants';
import { RelativeDatePipe } from '~/frontend/pipes/relative-date.pipe';
import {
  LinkFormDialogComponent,
  LinkFormDialogData,
} from '~/frontend/links/link-form-dialog/link-form-dialog.component';
import { buildShareUrl } from 'frontend/src/util/share-url';

export type LinkStatusFilter = 'active' | 'expired' | 'revoked';

@Component({
  selector: 'app-links-list',
  templateUrl: './links-list.component.html',
  styleUrl: './links-list.component.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    DatePipe,
    MatButtonModule,
    MatChipsModule,
    MatIconModule,
    MatMenuModule,
    MatTableModule,
    MatTooltipModule,
    RelativeDatePipe,
  ],
})
export class LinksListComponent {
  private readonly api = inject(Api);
  private readonly snackBar = inject(MatSnackBar);
  private readonly dialog = inject(MatDialog);
  private readonly clipboard = inject(Clipboard);

  protected readonly selectedTab = signal<LinkStatusFilter>('active');
  protected readonly refresh = signal(0);

  protected readonly listResource = resource({
    params: () => ({ status: this.selectedTab(), refresh: this.refresh() }),
    loader: ({ params }) => this.api.invoke(listLinks, { status: params.status }),
  });

  protected readonly visibleLinks = computed(() => this.listResource.value()?.items ?? []);

  private readonly revealedCodes = signal<ReadonlySet<string>>(new Set());

  protected isCodeRevealed(linkId: string): boolean {
    return this.revealedCodes().has(linkId);
  }

  protected toggleAndCopyCode(link: Link): void {
    if (!link.code) return;
    const wasRevealed = this.revealedCodes().has(link.id);
    this.revealedCodes.update((s) => {
      const next = new Set(s);
      if (wasRevealed) next.delete(link.id);
      else next.add(link.id);
      return next;
    });
    if (wasRevealed) return;
    if (this.clipboard.copy(link.code)) {
      this.snackBar.open('Code copied', 'OK', { duration: snackSuccessDuration });
    } else {
      this.snackBar.open('Failed to copy', 'Dismiss', { duration: snackErrorDuration });
    }
  }

  protected readonly dataSource = new MatTableDataSource<Link>([]);
  protected readonly displayedColumns = computed(() =>
    this.selectedTab() === 'revoked'
      ? ['icon', 'name', 'contents', 'opens', 'createdAt', 'expiresAt', 'code']
      : [
          'icon',
          'name',
          'contents',
          'opens',
          'createdAt',
          'expiresAt',
          'code',
          'shareUrl',
          'actions',
        ],
  );

  constructor() {
    effect(() => {
      this.dataSource.data = this.visibleLinks();
    });
  }

  protected onFilterChange(value: LinkStatusFilter): void {
    this.selectedTab.set(value);
  }

  protected shareUrl(link: Link): string {
    return buildShareUrl(link.key);
  }

  protected shareUrlDisplay(link: Link): string {
    return this.shareUrl(link).replace(/^https?:\/\//, '');
  }

  protected copyShareUrl(link: Link): void {
    if (this.clipboard.copy(this.shareUrl(link))) {
      this.snackBar.open('Share URL copied', 'OK', { duration: snackSuccessDuration });
    } else {
      this.snackBar.open('Failed to copy', 'Dismiss', { duration: snackErrorDuration });
    }
  }

  protected async openEditDialog(link: Link): Promise<void> {
    let detail;
    try {
      detail = await this.api.invoke(getLink, { id: link.id });
    } catch (e) {
      console.error(e);
      this.snackBar.open('Failed to open link', 'Dismiss', { duration: snackErrorDuration });
      return;
    }
    const ref = this.dialog.open(LinkFormDialogComponent, {
      data: { mode: 'edit', link: detail } satisfies LinkFormDialogData,
    });
    ref.afterClosed().subscribe((updated) => {
      if (updated) {
        this.refresh.update((v) => v + 1);
      }
    });
  }

  protected async regenerateLink(link: Link): Promise<void> {
    try {
      const updated = await this.api.invoke(rotateLinkKey, { id: link.id });
      this.refresh.update((v) => v + 1);
      const newUrl = buildShareUrl(updated.key);
      const snackRef = this.snackBar.open('Link regenerated', 'Copy new link', {
        duration: 6000,
      });
      snackRef.onAction().subscribe(() => {
        this.clipboard.copy(newUrl);
      });
    } catch (e) {
      console.error(e);
      this.snackBar.open('Failed to regenerate link', 'Dismiss', { duration: snackErrorDuration });
    }
  }

  protected async revoke(link: Link): Promise<void> {
    try {
      await this.api.invoke(revokeLink, { id: link.id });
      this.refresh.update((v) => v + 1);
      this.snackBar.open(`Revoked "${link.name}"`, 'OK', { duration: snackSuccessDuration });
    } catch (e) {
      console.error(e);
      this.snackBar.open('Failed to revoke link', 'Dismiss', { duration: snackErrorDuration });
    }
  }
}
