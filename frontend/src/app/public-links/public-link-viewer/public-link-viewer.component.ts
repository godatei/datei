import { HttpContext, HttpErrorResponse } from '@angular/common/http';
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
import { isPast } from 'date-fns';
import { Api } from '~/api/api';
import { downloadPublicLinkDatei, listPublicLinkDateien, unlockPublicLink } from '~/api/functions';
import type { Datei } from '~/api/models/datei';
import { RelativeDatePipe } from '~/frontend/pipes/relative-date.pipe';
import { PUBLIC_LINK_TOKEN } from '~/frontend/public-links/public-link-token.interceptor';
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
    RelativeDatePipe,
  ],
})
export class PublicLinkViewerComponent {
  private readonly route = inject(ActivatedRoute);
  private readonly api = inject(Api);
  private readonly snackBar = inject(MatSnackBar);

  private readonly paramMap = toSignal(this.route.paramMap);
  protected readonly key = computed(() => this.paramMap()?.get('key') ?? '');

  // Stored after a successful unlock. Re-used to re-unlock if the JWT expires
  // mid-session without bouncing the user back to the code prompt.
  private readonly code = signal<string>('');
  private readonly sessionToken = signal<string>('');

  // One-shot guard: when the link's root contains exactly one folder we
  // auto-descend so the viewer doesn't make people click through a trivial
  // wrapper. Only fires on first arrival — if the user later clicks the
  // breadcrumb back to root we respect that choice and stop here.
  private autoDescended = false;

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
  protected readonly isExpired = computed(() => {
    const date = this.expiresAt();
    return date !== null && isPast(date);
  });

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
          await this.unlockAndLoad(this.codeModel().code);
        },
      },
    },
  );

  constructor() {
    effect(() => {
      const key = this.key();
      if (key) {
        void this.unlockAndLoad();
      }
    });
  }

  // unlockAndLoad attempts an unlock (optionally with a candidate code) and,
  // on success, fetches the current folder. On 403 it transitions to the code
  // prompt.
  private async unlockAndLoad(candidateCode?: string): Promise<void> {
    if (!this.key()) return;
    try {
      const result = await this.api.invoke(unlockPublicLink, {
        key: this.key(),
        body: candidateCode !== undefined ? { code: candidateCode } : undefined,
      });
      this.sessionToken.set(result.token);
      if (candidateCode !== undefined) this.code.set(candidateCode);
    } catch (e) {
      this.handleUnlockError(e, candidateCode !== undefined);
      return;
    }
    await this.loadDateien(this.currentParentId());
    this.maybeAutoDescend();
  }

  // If the link's root contains exactly one folder, auto-navigate into it.
  // Runs at most once per page load and never when we're not at root.
  private maybeAutoDescend(): void {
    if (this.autoDescended) return;
    if (this.currentParentId() !== null) return;
    this.autoDescended = true;
    const items = this.dataSource.data;
    if (items.length === 1 && items[0].isDirectory) {
      void this.navigateInto(items[0]);
    }
  }

  private async loadDateien(parentId: string | null): Promise<void> {
    try {
      const result = await this.api.invoke(
        listPublicLinkDateien,
        { parentId: parentId ?? undefined },
        this.linkContext(),
      );
      this.dataSource.data = result.items;
      this.linkName.set(result.name);
      this.ownerName.set(result.ownerName);
      this.expiresAt.set(result.expiresAt ? new Date(result.expiresAt) : null);
      this.state.set({ kind: 'ready' });
    } catch (e) {
      if (e instanceof HttpErrorResponse && e.status === 401) {
        // Session JWT expired or the server restarted; transparently re-unlock
        // with the cached code (if any) so the user doesn't see a flicker.
        await this.unlockAndLoad(this.code() === '' ? undefined : this.code());
        return;
      }
      this.handleAccessError(e);
    }
  }

  private linkContext(): HttpContext {
    return new HttpContext().set(PUBLIC_LINK_TOKEN, this.sessionToken());
  }

  private handleUnlockError(e: unknown, fromSubmit: boolean): void {
    if (e instanceof HttpErrorResponse) {
      if (e.status === 403) {
        this.state.set({ kind: 'codeRequired', invalidCode: fromSubmit });
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

  // Errors from list/download after a successful unlock. 403 here means the
  // link's state changed (revoked / expired / out-of-scope datei), not "code
  // missing"; the unlock path is the only place that produces a code-prompt.
  private handleAccessError(e: unknown): void {
    if (e instanceof HttpErrorResponse) {
      if (e.status === 403) {
        this.state.set({ kind: 'expired' });
        return;
      }
      if (e.status === 404) {
        this.state.set({ kind: 'notFound' });
        return;
      }
    }
    console.error(e);
    this.state.set({ kind: 'error', message: 'Failed to load shared files' });
  }

  protected async navigateInto(folder: Datei): Promise<void> {
    if (!folder.isDirectory) return;
    this.path.update((p) => [...p, { id: folder.id, name: folder.name ?? 'Folder' }]);
    await this.loadDateien(folder.id);
  }

  protected async navigateToBreadcrumb(index: number): Promise<void> {
    this.path.update((p) => p.slice(0, index + 1));
    await this.loadDateien(this.currentParentId());
  }

  protected async download(item: Datei): Promise<void> {
    if (item.isDirectory) {
      void this.navigateInto(item);
      return;
    }
    await this.downloadWithRetry(item, false);
  }

  private async downloadWithRetry(item: Datei, retried: boolean): Promise<void> {
    try {
      const response = await this.api.invoke$Response(
        downloadPublicLinkDatei,
        { dateiId: item.id },
        this.linkContext(),
      );
      triggerDownload(response.body as unknown as Blob, item.name ?? 'download');
    } catch (e) {
      if (e instanceof HttpErrorResponse && e.status === 401 && !retried) {
        // Session JWT expired mid-session — re-unlock with the cached code (if
        // any) and retry once. The `retried` guard prevents loops if the
        // re-unlock also returns a token the server immediately rejects.
        await this.unlockAndLoad(this.code() === '' ? undefined : this.code());
        if (this.sessionToken()) {
          await this.downloadWithRetry(item, true);
        }
        return;
      }
      if (e instanceof HttpErrorResponse && (e.status === 403 || e.status === 404)) {
        this.handleAccessError(e);
        return;
      }
      console.error(e);
      this.snackBar.open('Failed to download', 'Dismiss', { duration: 4000 });
    }
  }

  protected readonly formatBytes = formatBytes;
}
