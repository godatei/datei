import { DatePipe } from '@angular/common';
import { SelectionModel } from '@angular/cdk/collections';
import {
  Component,
  computed,
  DestroyRef,
  effect,
  inject,
  resource,
  signal,
  untracked,
} from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { MatButtonModule } from '@angular/material/button';
import { MatDialog } from '@angular/material/dialog';
import { MatIconModule } from '@angular/material/icon';
import { MatMenuModule } from '@angular/material/menu';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatTableDataSource, MatTableModule } from '@angular/material/table';
import { ActivatedRoute, Router } from '@angular/router';
import { DomSanitizer } from '@angular/platform-browser';
import { Api } from 'frontend/src/api/api';
import {
  createDatei,
  deleteDatei,
  downloadDatei,
  getDateiPath,
  listDatei,
  updateDatei$FormData,
} from 'frontend/src/api/functions';
import { Datei } from 'frontend/src/api/models';
import {
  ImagePreviewDialogComponent,
  ImagePreviewDialogData,
} from './image-preview-dialog.component';
import { NewFolderDialogComponent } from './new-folder-dialog.component';

@Component({
  selector: 'app-dashboard',
  templateUrl: './dashboard.component.html',
  styleUrls: ['./dashboard.component.css'],
  imports: [MatMenuModule, MatIconModule, MatButtonModule, MatTableModule, DatePipe],
})
export class DashboardComponent {
  private readonly api = inject(Api);
  private readonly snackBar = inject(MatSnackBar);
  private readonly dialog = inject(MatDialog);
  private readonly sanitizer = inject(DomSanitizer);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);

  private readonly refresh = signal(0);
  private readonly queryParams = toSignal(this.route.queryParamMap);

  protected readonly parentId = computed(() => this.queryParams()?.get('parentId') ?? null);

  protected readonly listDateiResource = resource({
    params: () => ({ parentId: this.parentId(), refresh: this.refresh() }),
    loader: ({ params }) =>
      this.api.invoke(listDatei, params.parentId ? { parentId: params.parentId } : undefined),
  });

  protected readonly pathResource = resource({
    params: () => ({ parentId: this.parentId() }),
    loader: ({ params }) =>
      params.parentId
        ? this.api.invoke(getDateiPath, { id: params.parentId })
        : Promise.resolve([]),
  });
  protected readonly dataSource = new MatTableDataSource<Datei>([]);
  protected readonly displayedColumns = ['name', 'createdAt', 'updatedAt', 'mimeType', 'actions'];
  protected readonly selection = new SelectionModel<Datei>(true, [], true, (a, b) => a.id === b.id);
  protected readonly selectedIds = signal<ReadonlySet<string>>(new Set());
  protected readonly uploading = signal(false);
  protected readonly isDragging = signal(false);
  protected readonly dragOverDirectoryId = signal<string | null>(null);
  protected readonly dragPointerPos = signal({ x: 0, y: 0 });
  private selectionAnchor: Datei | null = null;
  protected dragItems: Datei[] = [];
  private dragStartPos: { x: number; y: number } | null = null;
  private dragStartRow: Datei | null = null;
  private dragJustOccurred = false;
  private readonly destroyRef = inject(DestroyRef);
  private readonly onPointerMoveBound = this.onPointerMove.bind(this);
  private readonly onPointerUpBound = this.onPointerUp.bind(this);

  constructor() {
    effect(() => {
      this.dataSource.data = this.listDateiResource.value()?.items ?? [];
      untracked(() => {
        this.selection.clear();
        this.selectionAnchor = null;
      });
    });
    const sub = this.selection.changed.subscribe(() => {
      this.selectedIds.set(new Set(this.selection.selected.map((d) => d.id)));
    });
    this.destroyRef.onDestroy(() => {
      sub.unsubscribe();
      document.removeEventListener('mousemove', this.onPointerMoveBound);
      document.removeEventListener('mouseup', this.onPointerUpBound);
      document.body.style.cursor = '';
    });
  }

  protected onRowClick(row: Datei, event: MouseEvent): void {
    if (this.dragJustOccurred) {
      this.dragJustOccurred = false;
      return;
    }
    if (event.shiftKey && this.selectionAnchor !== null) {
      const data = this.dataSource.data;
      const anchorIdx = data.findIndex((d) => d.id === this.selectionAnchor!.id);
      const rowIdx = data.findIndex((d) => d.id === row.id);
      if (anchorIdx !== -1 && rowIdx !== -1) {
        const [lo, hi] = anchorIdx <= rowIdx ? [anchorIdx, rowIdx] : [rowIdx, anchorIdx];
        this.selection.clear();
        data.slice(lo, hi + 1).forEach((d) => this.selection.select(d));
      }
    } else if (event.ctrlKey || event.metaKey) {
      this.selection.toggle(row);
      this.selectionAnchor = row;
    } else {
      this.selection.clear();
      this.selection.select(row);
      this.selectionAnchor = row;
    }
  }

  protected onRowDblClick(row: Datei): void {
    if (row.isDirectory) {
      this.selection.clear();
      this.selectionAnchor = null;
      this.router.navigate([], { relativeTo: this.route, queryParams: { parentId: row.id } });
      return;
    }
    const name = row.name ?? '';
    if (row.mimeType?.startsWith('image/')) {
      void this.previewImage(row.id, name);
    } else {
      void this.downloadFile(row.id, name);
    }
  }

  private async previewImage(id: string, name: string): Promise<void> {
    try {
      const response = await this.api.invoke$Response(downloadDatei, { id });
      const url = URL.createObjectURL(response.body as Blob);
      const ref = this.dialog.open(ImagePreviewDialogComponent, {
        data: {
          src: this.sanitizer.bypassSecurityTrustUrl(url),
          name,
        } satisfies ImagePreviewDialogData,
        maxWidth: '90vw',
        maxHeight: '90vh',
      });
      ref.afterClosed().subscribe(() => URL.revokeObjectURL(url));
    } catch (e) {
      console.error(e);
      this.snackBar.open('Failed to load image', 'Dismiss', { duration: 4000 });
    }
  }

  private async downloadFile(id: string, name: string): Promise<void> {
    try {
      const response = await this.api.invoke$Response(downloadDatei, { id });
      const url = URL.createObjectURL(response.body as Blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = name;
      a.click();
      URL.revokeObjectURL(url);
    } catch (e) {
      console.error(e);
      this.snackBar.open('Failed to download file', 'Dismiss', { duration: 4000 });
    }
  }

  protected navigateTo(id: string | null): void {
    this.router.navigate([], {
      relativeTo: this.route,
      queryParams: id ? { parentId: id } : {},
    });
  }

  protected openNewFolderDialog(): void {
    const ref = this.dialog.open(NewFolderDialogComponent, { width: '360px' });
    ref.afterClosed().subscribe(async (name: string | null) => {
      if (!name) return;
      try {
        const parentId = this.parentId() ?? undefined;
        await this.api.invoke(createDatei, { body: { name, parentId } });
        this.refresh.update((v) => v + 1);
      } catch (e) {
        console.error(e);
        this.snackBar.open('Failed to create folder', 'Dismiss', { duration: 4000 });
      }
    });
  }

  protected async trashDatei(id: string, event: Event): Promise<void> {
    event.stopPropagation();
    try {
      await this.api.invoke(deleteDatei, { id });
      this.refresh.update((v) => v + 1);
    } catch (e) {
      console.error(e);
      this.snackBar.open('Failed to move to trash', 'Dismiss', { duration: 4000 });
    }
  }

  protected onRowMouseDown(row: Datei, event: MouseEvent): void {
    if (event.button !== 0) return;
    this.dragStartPos = { x: event.clientX, y: event.clientY };
    this.dragStartRow = row;
    document.addEventListener('mousemove', this.onPointerMoveBound);
    document.addEventListener('mouseup', this.onPointerUpBound);
  }

  private onPointerMove(event: MouseEvent): void {
    if (!this.dragStartPos || !this.dragStartRow) return;

    if (!this.isDragging()) {
      const dx = event.clientX - this.dragStartPos.x;
      const dy = event.clientY - this.dragStartPos.y;
      if (Math.hypot(dx, dy) < 5) return;

      if (!this.selection.isSelected(this.dragStartRow)) {
        this.selection.clear();
        this.selection.select(this.dragStartRow);
      }
      this.dragItems = [...this.selection.selected];
      this.isDragging.set(true);
    }

    this.dragPointerPos.set({ x: event.clientX, y: event.clientY });

    const el = document.elementFromPoint(event.clientX, event.clientY);
    const target = el?.closest<HTMLElement>('[data-drop-target]');
    if (target) {
      const id = target.dataset['dropTarget']!;
      if (!this.dragItems.some((d) => d.id === id)) {
        this.dragOverDirectoryId.set(id);
        return;
      }
    }
    this.dragOverDirectoryId.set(null);
  }

  private async onPointerUp(): Promise<void> {
    document.removeEventListener('mousemove', this.onPointerMoveBound);
    document.removeEventListener('mouseup', this.onPointerUpBound);

    const targetId = this.dragOverDirectoryId();
    const wasDragging = this.isDragging();

    this.isDragging.set(false);
    this.dragOverDirectoryId.set(null);
    this.dragStartPos = null;
    this.dragStartRow = null;

    if (wasDragging) {
      // Suppress the synthetic click that the browser fires after mouseup on the same element.
      this.dragJustOccurred = true;
      setTimeout(() => {
        this.dragJustOccurred = false;
      }, 0);
    }
    if (!wasDragging || targetId === null) return;

    // targetId === '' means move to root; pass as empty string so the multipart
    // endpoint interprets it as "move to root" (no UUID → newParentID = nil).
    const items = this.dragItems;
    const results = await Promise.allSettled(
      items.map((item) =>
        this.api.invoke(updateDatei$FormData, {
          id: item.id,
          body: { parentId: targetId },
        }),
      ),
    );

    const failed = results.filter((r) => r.status === 'rejected').length;
    if (failed > 0) {
      this.snackBar.open(`Failed to move ${failed} item(s)`, 'Dismiss', { duration: 4000 });
    }
    this.refresh.update((v) => v + 1);
  }

  protected async startUpload(el: HTMLInputElement) {
    if (el.files === null || el.files.length === 0) {
      return;
    }

    const snack = this.snackBar.open('Upload in progress…');
    this.uploading.set(true);
    try {
      const file = el.files[0];
      const parentId = this.parentId() ?? undefined;
      await this.api.invoke(createDatei, { body: { file, parentId } });
      this.refresh.update((v) => v + 1);
    } catch (e) {
      console.error(e);
    } finally {
      setTimeout(() => snack.dismiss(), 500);
      this.uploading.set(false);
    }
  }
}
