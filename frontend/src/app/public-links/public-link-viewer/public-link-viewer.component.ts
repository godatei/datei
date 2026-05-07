import { HttpErrorResponse } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  inject,
  signal,
} from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { form, FormField, FormRoot, required } from '@angular/forms/signals';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatTableDataSource, MatTableModule } from '@angular/material/table';
import { ActivatedRoute } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import type { Datei } from '~/api/models/datei';
import { LinksService } from '~/frontend/services/links.service';
import { triggerDownload } from 'frontend/src/util/download';
import { formatBytes } from 'frontend/src/util/format-bytes';

type ViewerState =
  | { kind: 'loading' }
  | { kind: 'codeRequired'; invalidCode?: boolean }
  | { kind: 'ready' }
  | { kind: 'expired' }
  | { kind: 'notFound' }
  | { kind: 'error'; message: string };

@Component({
  selector: 'app-public-link-viewer',
  templateUrl: './public-link-viewer.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatButtonModule,
    MatFormFieldModule,
    MatIconModule,
    MatInputModule,
    MatTableModule,
    FormField,
    FormRoot,
  ],
})
export class PublicLinkViewerComponent {
  private readonly route = inject(ActivatedRoute);
  private readonly linksService = inject(LinksService);
  private readonly snackBar = inject(MatSnackBar);

  private readonly paramMap = toSignal(this.route.paramMap);
  protected readonly accessToken = computed(() => this.paramMap()?.get('accessToken') ?? '');

  private readonly code = signal<string>('');
  protected readonly state = signal<ViewerState>({ kind: 'loading' });
  protected readonly invalidCode = computed(() => {
    const s = this.state();
    return s.kind === 'codeRequired' && s.invalidCode === true;
  });
  protected readonly stateMessage = computed(() => {
    const s = this.state();
    return s.kind === 'error' ? s.message : '';
  });
  protected readonly dataSource = new MatTableDataSource<Datei>([]);
  protected readonly linkName = signal<string>('');
  protected readonly ownerName = signal<string>('');
  protected readonly expiresAt = signal<Date | null>(null);
  protected readonly expiryText = computed(() => formatExpiryText(this.expiresAt()));

  protected readonly displayedColumns = ['icon', 'name', 'size', 'actions'];
  protected readonly path = signal<{ id: string | null; name: string }[]>([
    { id: null, name: 'Shared files' },
  ]);
  protected readonly currentParentId = computed(() => {
    const p = this.path();
    return p[p.length - 1]?.id ?? null;
  });

  protected readonly codeModel = signal({ code: '' });
  protected readonly codeForm = form(
    this.codeModel,
    (p) => {
      required(p.code);
    },
    {
      submission: {
        action: async () => {
          this.code.set(this.codeModel().code);
          this.state.set({ kind: 'loading' });
          await this.loadDateien(this.currentParentId());
        },
      },
    },
  );

  constructor() {
    effect(() => {
      const token = this.accessToken();
      if (token) {
        void this.loadDateien(null);
      }
    });
  }

  private async loadDateien(parentId: string | null): Promise<void> {
    const token = this.accessToken();
    if (!token) return;
    try {
      const result = await firstValueFrom(
        this.linksService.listPublicDateien(
          token,
          parentId ?? undefined,
          this.code() === '' ? undefined : this.code(),
        ),
      );
      this.dataSource.data = result.items;
      this.linkName.set(result.name);
      this.ownerName.set(result.ownerName);
      this.expiresAt.set(result.expiresAt ? new Date(result.expiresAt) : null);
      this.state.set({ kind: 'ready' });
    } catch (e) {
      this.handleError(e);
    }
  }

  private handleError(e: unknown): void {
    if (e instanceof HttpErrorResponse) {
      if (e.status === 403) {
        const code = (e.error?.code as string | undefined) ?? '';
        const invalid = code === 'code_invalid';
        this.state.set({ kind: 'codeRequired', invalidCode: invalid });
        return;
      }
      if (e.status === 404) {
        this.state.set({ kind: 'notFound' });
        return;
      }
      if (e.status === 410) {
        this.state.set({ kind: 'expired' });
        return;
      }
    }
    console.error(e);
    this.state.set({ kind: 'error', message: 'Failed to load shared files' });
  }

  protected async navigateInto(folder: Datei): Promise<void> {
    if (!folder.isDirectory) return;
    this.path.update((p) => [...p, { id: folder.id, name: folder.name ?? 'Folder' }]);
    this.state.set({ kind: 'loading' });
    await this.loadDateien(folder.id);
  }

  protected async navigateToBreadcrumb(index: number): Promise<void> {
    this.path.update((p) => p.slice(0, index + 1));
    this.state.set({ kind: 'loading' });
    await this.loadDateien(this.currentParentId());
  }

  protected async download(item: Datei): Promise<void> {
    if (item.isDirectory) {
      void this.navigateInto(item);
      return;
    }
    const token = this.accessToken();
    if (!token) return;
    try {
      const blob = await firstValueFrom(
        this.linksService.downloadPublicDatei(
          token,
          item.id,
          this.code() === '' ? undefined : this.code(),
        ),
      );
      triggerDownload(blob, item.name ?? 'download');
    } catch (e) {
      if (
        e instanceof HttpErrorResponse &&
        (e.status === 403 || e.status === 410 || e.status === 404)
      ) {
        this.handleError(e);
        return;
      }
      console.error(e);
      this.snackBar.open('Failed to download', 'Dismiss', { duration: 4000 });
    }
  }

  protected readonly formatBytes = formatBytes;
}

function formatExpiryText(expiresAt: Date | null): string {
  if (!expiresAt) return '';
  const ms = expiresAt.getTime() - Date.now();
  if (ms <= 0) return 'Expired';
  const minutes = Math.floor(ms / 60_000);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);
  if (days >= 30) return `Expires on ${expiresAt.toLocaleDateString()}`;
  if (days >= 1) return `Expires in ${days} ${days === 1 ? 'day' : 'days'}`;
  if (hours >= 1) return `Expires in ${hours} ${hours === 1 ? 'hour' : 'hours'}`;
  if (minutes >= 1) return `Expires in ${minutes} ${minutes === 1 ? 'minute' : 'minutes'}`;
  return 'Expires in less than a minute';
}
