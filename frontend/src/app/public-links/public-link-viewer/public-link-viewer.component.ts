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
import { isPast } from 'date-fns';
import { firstValueFrom } from 'rxjs';
import type { Datei } from '~/api/models/datei';
import { LinksService } from '~/frontend/services/links.service';
import { RelativeDatePipe } from '~/frontend/pipes/relative-date.pipe';
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
          await this.loadDateien(this.currentParentId(), this.codeModel().code);
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

  private async loadDateien(parentId: string | null, candidateCode?: string): Promise<void> {
    const token = this.accessToken();
    if (!token) return;
    const code = candidateCode ?? (this.code() === '' ? undefined : this.code());
    try {
      const result = await firstValueFrom(
        this.linksService.listPublicDateien(token, parentId ?? undefined, code),
      );
      if (candidateCode !== undefined) this.code.set(candidateCode);
      this.dataSource.data = result.items;
      this.linkName.set(result.name);
      this.ownerName.set(result.ownerName);
      this.expiresAt.set(result.expiresAt ? new Date(result.expiresAt) : null);
      this.state.set({ kind: 'ready' });
    } catch (e) {
      this.handleError(e, candidateCode !== undefined);
    }
  }

  private handleError(e: unknown, fromSubmit = false): void {
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
