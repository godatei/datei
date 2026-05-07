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
import { firstValueFrom } from 'rxjs';
import type { Link } from '~/api/models/link';
import { RelativeDatePipe } from '~/frontend/pipes/relative-date.pipe';
import { LinksService, type LinkStatusFilter } from '~/frontend/services/links.service';
import {
  LinkFormDialogComponent,
  LinkFormDialogData,
} from '~/frontend/links/link-form-dialog/link-form-dialog.component';

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
  private readonly linksService = inject(LinksService);
  private readonly snackBar = inject(MatSnackBar);
  private readonly dialog = inject(MatDialog);
  private readonly clipboard = inject(Clipboard);

  protected readonly selectedTab = signal<LinkStatusFilter>('active');
  protected readonly refresh = signal(0);

  protected readonly listResource = resource({
    params: () => ({ status: this.selectedTab(), refresh: this.refresh() }),
    loader: ({ params }) => firstValueFrom(this.linksService.listLinks({ status: params.status })),
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
      this.snackBar.open('Code copied', 'OK', { duration: 2000 });
    } else {
      this.snackBar.open('Failed to copy', 'Dismiss', { duration: 3000 });
    }
  }

  protected readonly dataSource = new MatTableDataSource<Link>([]);
  protected readonly displayedColumns = computed(() =>
    this.selectedTab() === 'revoked'
      ? ['icon', 'name', 'contents', 'createdAt', 'expiresAt', 'code']
      : ['icon', 'name', 'contents', 'createdAt', 'expiresAt', 'code', 'shareUrl', 'actions'],
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
    return this.linksService.buildShareUrl(link.accessToken);
  }

  protected shareUrlDisplay(link: Link): string {
    return this.shareUrl(link).replace(/^https?:\/\//, '');
  }

  protected copyShareUrl(link: Link): void {
    if (this.clipboard.copy(this.shareUrl(link))) {
      this.snackBar.open('Share URL copied', 'OK', { duration: 2000 });
    } else {
      this.snackBar.open('Failed to copy', 'Dismiss', { duration: 3000 });
    }
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
