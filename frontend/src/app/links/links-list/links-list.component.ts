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
import { MatDialog } from '@angular/material/dialog';
import { MatIconModule } from '@angular/material/icon';
import { MatMenuModule } from '@angular/material/menu';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatTableDataSource, MatTableModule } from '@angular/material/table';
import { MatTabsModule } from '@angular/material/tabs';
import { MatTooltipModule } from '@angular/material/tooltip';
import { RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import type { Link } from '~/api/models/link';
import { LinksService } from '~/frontend/services/links.service';
import {
  LinkFormDialogComponent,
  LinkFormDialogData,
} from '~/frontend/links/link-form-dialog/link-form-dialog.component';

export type LinkStatus = 'active' | 'expired' | 'revoked';

export function statusOf(link: Link): LinkStatus {
  if (link.revokedAt) return 'revoked';
  if (link.expiresAt && new Date(link.expiresAt).getTime() <= Date.now()) return 'expired';
  return 'active';
}

@Component({
  selector: 'app-links-list',
  templateUrl: './links-list.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    DatePipe,
    MatButtonModule,
    MatIconModule,
    MatMenuModule,
    MatTableModule,
    MatTabsModule,
    MatTooltipModule,
    RouterLink,
  ],
})
export class LinksListComponent {
  private readonly linksService = inject(LinksService);
  private readonly snackBar = inject(MatSnackBar);
  private readonly dialog = inject(MatDialog);
  private readonly clipboard = inject(Clipboard);

  protected readonly refresh = signal(0);
  protected readonly listResource = resource({
    params: () => ({ refresh: this.refresh() }),
    loader: () => firstValueFrom(this.linksService.listLinks()),
  });

  private readonly allLinks = computed(() => this.listResource.value() ?? []);
  protected readonly activeLinks = computed(() =>
    this.allLinks().filter((l) => statusOf(l) === 'active'),
  );
  protected readonly expiredLinks = computed(() =>
    this.allLinks().filter((l) => statusOf(l) === 'expired'),
  );
  protected readonly revokedLinks = computed(() =>
    this.allLinks().filter((l) => statusOf(l) === 'revoked'),
  );

  protected readonly selectedTab = signal<LinkStatus>('active');
  protected readonly selectedTabIndex = computed(() => {
    switch (this.selectedTab()) {
      case 'active':
        return 0;
      case 'expired':
        return 1;
      case 'revoked':
        return 2;
    }
  });
  protected readonly allLinksTotal = computed(() => this.allLinks().length);
  private readonly visibleLinks = computed(() => {
    switch (this.selectedTab()) {
      case 'active':
        return this.activeLinks();
      case 'expired':
        return this.expiredLinks();
      case 'revoked':
        return this.revokedLinks();
    }
  });

  private readonly revealedCodes = signal<ReadonlySet<string>>(new Set());

  protected isCodeRevealed(linkId: string): boolean {
    return this.revealedCodes().has(linkId);
  }

  protected toggleCodeVisibility(linkId: string): void {
    this.revealedCodes.update((s) => {
      const next = new Set(s);
      if (next.has(linkId)) next.delete(linkId);
      else next.add(linkId);
      return next;
    });
  }

  protected readonly dataSource = new MatTableDataSource<Link>([]);
  protected readonly displayedColumns = computed(() =>
    this.selectedTab() === 'revoked'
      ? ['name', 'contents', 'createdAt', 'expiresAt', 'code']
      : ['name', 'contents', 'createdAt', 'expiresAt', 'code', 'shareUrl', 'actions'],
  );

  constructor() {
    effect(() => {
      this.dataSource.data = this.visibleLinks();
    });
  }

  protected onTabChange(index: number): void {
    const tab: LinkStatus = index === 1 ? 'expired' : index === 2 ? 'revoked' : 'active';
    this.selectedTab.set(tab);
  }

  protected shareUrl(link: Link): string {
    return this.linksService.buildShareUrl(link.accessToken);
  }

  protected copyShareUrl(link: Link): void {
    if (this.clipboard.copy(this.shareUrl(link))) {
      this.snackBar.open('Share URL copied', 'OK', { duration: 2000 });
    } else {
      this.snackBar.open('Failed to copy', 'Dismiss', { duration: 3000 });
    }
  }

  protected copyCode(link: Link): void {
    if (!link.code) return;
    if (this.clipboard.copy(link.code)) {
      this.snackBar.open('Code copied', 'OK', { duration: 2000 });
    } else {
      this.snackBar.open('Failed to copy', 'Dismiss', { duration: 3000 });
    }
  }

  protected openPreview(link: Link): void {
    window.open(this.shareUrl(link), '_blank', 'noopener');
  }

  protected async openEditDialog(link: Link): Promise<void> {
    let detail;
    try {
      detail = await firstValueFrom(this.linksService.getLink(link.id));
    } catch (e) {
      console.error(e);
      this.snackBar.open('Failed to open link', 'Dismiss', { duration: 4000 });
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

  protected async rotateAccessToken(link: Link): Promise<void> {
    try {
      const updated = await firstValueFrom(this.linksService.rotateAccessToken(link.id));
      this.refresh.update((v) => v + 1);
      const newUrl = this.linksService.buildShareUrl(updated.accessToken);
      const snackRef = this.snackBar.open('Access token rotated', 'Copy new link', {
        duration: 6000,
      });
      snackRef.onAction().subscribe(() => {
        this.clipboard.copy(newUrl);
      });
    } catch (e) {
      console.error(e);
      this.snackBar.open('Failed to rotate access token', 'Dismiss', { duration: 4000 });
    }
  }

  protected async revokeLink(link: Link): Promise<void> {
    try {
      await firstValueFrom(this.linksService.revokeLink(link.id));
      this.refresh.update((v) => v + 1);
      this.snackBar.open(`Revoked "${link.name}"`, 'OK', { duration: 3000 });
    } catch (e) {
      console.error(e);
      this.snackBar.open('Failed to revoke link', 'Dismiss', { duration: 4000 });
    }
  }
}
