import { HttpContext, HttpErrorResponse } from '@angular/common/http';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  effect,
  inject,
  signal,
} from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { form, FormField, FormRoot, pattern, required } from '@angular/forms/signals';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatTableDataSource, MatTableModule } from '@angular/material/table';
import { DomSanitizer, SafeUrl } from '@angular/platform-browser';
import { ActivatedRoute } from '@angular/router';
import { isPast } from 'date-fns';
import { Api } from '~/api/api';
import { downloadPublicLinkDatei, listPublicLinkDateien, unlockPublicLink } from '~/api/functions';
import type { Datei } from '~/api/models/datei';
import { ImagePreviewComponent } from '~/frontend/components/image-preview.component';
import { BytesPipe } from '~/frontend/pipes/bytes.pipe';
import { RelativeDatePipe } from '~/frontend/pipes/relative-date.pipe';
import { PUBLIC_LINK_TOKEN } from '~/frontend/public-links/public-link-token.interceptor';
import { triggerDownload } from '~/util/download';

type ViewerState =
  | { kind: 'loading' }
  | { kind: 'codeRequired'; invalidCode?: boolean }
  | { kind: 'ready' }
  | { kind: 'expired' }
  | { kind: 'unavailable' }
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
    BytesPipe,
    ImagePreviewComponent,
    RelativeDatePipe,
  ],
})
export class PublicLinkViewerComponent {
  private readonly route = inject(ActivatedRoute);
  private readonly api = inject(Api);
  private readonly snackBar = inject(MatSnackBar);
  private readonly sanitizer = inject(DomSanitizer);

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
  private readonly items = signal<Datei[]>([]);
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

  // singleFile is set when the link's root contains exactly one non-directory
  // item; the viewer renders a focused file card instead of the table.
  protected readonly singleFile = computed(() => {
    if (this.currentParentId() !== null) return null;
    const items = this.items();
    if (items.length === 1 && !items[0].isDirectory) return items[0];
    return null;
  });

  // Object URL + SafeUrl for the inline image preview. The blob itself is
  // retained so the Download button can hand it to triggerDownload without a
  // second network round-trip.
  private previewObjectUrl: string | null = null;
  private previewBlob: Blob | null = null;
  protected readonly previewUrl = signal<SafeUrl | null>(null);

  protected readonly codeModel = signal({ code: '' });
  protected readonly codeForm = form(
    this.codeModel,
    (p) => {
      required(p.code);
      pattern(p.code, /\S/);
    },
    {
      submission: {
        action: async () => {
          await this.unlockAndLoad(this.codeModel().code.trim());
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
    // Lazily load the inline image preview when the link resolves to a single
    // image file. Effect re-runs are no-op'd by the `previewBlob` guard.
    effect(() => {
      const file = this.singleFile();
      if (file && isImageMime(file.mimeType)) {
        void this.loadSinglePreview(file);
      } else if (!file) {
        this.clearPreview();
      }
    });
    inject(DestroyRef).onDestroy(() => this.clearPreview());
  }

  private async unlockAndLoad(candidateCode?: string, retried = false): Promise<void> {
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
    await this.loadDateien(this.currentParentId(), retried);
    this.maybeAutoDescend();
  }

  private maybeAutoDescend(): void {
    if (this.autoDescended) return;
    if (this.state().kind !== 'ready') return;
    if (this.currentParentId() !== null) return;
    this.autoDescended = true;
    const items = this.items();
    if (items.length === 1 && items[0].isDirectory) {
      void this.navigateInto(items[0]);
    }
  }

  private async loadDateien(parentId: string | null, retried = false): Promise<void> {
    try {
      const result = await this.api.invoke(
        listPublicLinkDateien,
        { parentId: parentId ?? undefined },
        this.linkContext(),
      );
      this.items.set(result.items);
      this.dataSource.data = result.items;
      this.linkName.set(result.name);
      this.ownerName.set(result.ownerName);
      this.expiresAt.set(result.expiresAt ? new Date(result.expiresAt) : null);
      this.state.set({ kind: 'ready' });
    } catch (e) {
      if (e instanceof HttpErrorResponse && e.status === 401 && !retried) {
        await this.unlockAndLoad(this.code() === '' ? undefined : this.code(), true);
        return;
      }
      this.handleAccessError(e);
    }
  }

  private async loadSinglePreview(file: Datei): Promise<void> {
    if (this.previewBlob) return;
    try {
      const response = await this.api.invoke$Response(
        downloadPublicLinkDatei,
        { dateiId: file.id },
        this.linkContext(),
      );
      const blob = response.body as unknown as Blob;
      const url = URL.createObjectURL(blob);
      this.previewBlob = blob;
      this.previewObjectUrl = url;
      this.previewUrl.set(this.sanitizer.bypassSecurityTrustUrl(url));
    } catch (e) {
      // Preview is best-effort; the user can still click Download to get the
      // file. We swallow errors here so a failed preview load doesn't poison
      // the rest of the page state.
      console.error(e);
    }
  }

  private clearPreview(): void {
    if (this.previewObjectUrl) {
      URL.revokeObjectURL(this.previewObjectUrl);
      this.previewObjectUrl = null;
    }
    this.previewBlob = null;
    this.previewUrl.set(null);
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
      // Unlock returns 404 for not-found, revoked, and expired — the viewer
      // doesn't try to differentiate (info-hiding posture).
      if (e.status === 404) {
        this.state.set({ kind: 'notFound' });
        return;
      }
    }
    console.error(e);
    this.state.set({ kind: 'error', message: 'Failed to load shared files' });
  }

  private handleAccessError(e: unknown): void {
    if (e instanceof HttpErrorResponse) {
      if (e.status === 403) {
        this.state.set({ kind: this.isExpired() ? 'expired' : 'unavailable' });
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
    // Reuse the preview blob if we already fetched it for inline rendering.
    if (this.previewBlob && this.singleFile()?.id === item.id) {
      triggerDownload(this.previewBlob, item.name ?? 'download');
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
}

function isImageMime(mime: string | null | undefined): boolean {
  return mime != null && mime.startsWith('image/');
}
