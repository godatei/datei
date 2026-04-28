import { DatePipe } from '@angular/common';
import { SelectionModel } from '@angular/cdk/collections';
import {
  Component,
  computed,
  effect,
  inject,
  resource,
  signal,
} from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { map } from 'rxjs';
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
import { DragDropDirective, DropEvent } from './drag-drop.directive';
import { DragPreviewDirective } from './drag-preview.directive';

@Component({
  selector: 'app-dashboard',
  templateUrl: './dashboard.component.html',
  styleUrls: ['./dashboard.component.css'],
  imports: [MatMenuModule, MatIconModule, MatButtonModule, MatTableModule, DatePipe, DragDropDirective, DragPreviewDirective],
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
  protected readonly selectedIds = toSignal(
    this.selection.changed.pipe(map(() => new Set(this.selection.selected.map((d) => d.id)))),
    { initialValue: new Set<string>() },
  );
  protected readonly uploading = signal(false);
  private selectionAnchor: Datei | null = null;

  constructor() {
    effect(() => {
      this.dataSource.data = this.listDateiResource.value()?.items ?? [];
      this.selection.clear();
      this.selectionAnchor = null;
    });
  }

  protected onRowClick(row: Datei, event: MouseEvent): void {
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

  protected async onDrop(event: DropEvent): Promise<void> {
    // targetId === '' means move to root; the multipart endpoint interprets
    // an empty string as "no UUID → newParentID = nil".
    const results = await Promise.allSettled(
      event.items.map((item) =>
        this.api.invoke(updateDatei$FormData, {
          id: item.id,
          body: { parentId: event.targetId },
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
