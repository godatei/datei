import { Clipboard } from '@angular/cdk/clipboard';
import { DatePipe } from '@angular/common';
import {
  Component,
  computed,
  effect,
  inject,
  resource,
  signal,
  viewChild,
  ChangeDetectionStrategy,
} from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { MatButtonModule } from '@angular/material/button';
import { MatChipsModule } from '@angular/material/chips';
import { MatDialog } from '@angular/material/dialog';
import { MatIconModule } from '@angular/material/icon';
import { MatMenuModule } from '@angular/material/menu';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatTableDataSource, MatTableModule } from '@angular/material/table';
import { MatTooltipModule } from '@angular/material/tooltip';
import { ActivatedRoute, Router } from '@angular/router';
import { DomSanitizer } from '@angular/platform-browser';
import { Api } from '~/api/api';
import {
  addDateiToLink,
  createDatei,
  deleteDatei,
  downloadDatei,
  getDateiPath,
  listDatei,
  updateDatei$FormData,
} from '~/api/functions';
import { Datei } from '~/api/models';
import { ThumbnailIconComponent } from './thumbnail-icon.component';
import {
  ImagePreviewDialogComponent,
  ImagePreviewDialogData,
} from './image-preview-dialog.component';
import { NewFolderDialogComponent } from './new-folder-dialog.component';
import { RenameDateiDialogComponent, RenameDateiDialogData } from './rename-datei-dialog.component';
import {
  LinkFormDialogComponent,
  LinkFormDialogData,
} from '~/frontend/links/link-form-dialog/link-form-dialog.component';
import { LinkPickerDialogComponent } from '~/frontend/links/link-picker-dialog/link-picker-dialog.component';
import type { Link } from '~/api/models/link';
import { BytesPipe } from '~/frontend/pipes/bytes.pipe';
import { triggerDownload } from '~/util/download';
import { buildShareUrl } from '~/util/share-url';
import { DragDropDirective, DropEvent } from './drag-drop.directive';
import { DragPreviewDirective } from './drag-preview.directive';
import { DragItemDirective } from './drag-row.directive';
import { DropTargetDirective } from './drop-target.directive';
import { SelectionDirective } from './selection.directive';
import { SelectionItemDirective } from './selection-item.directive';
import { snackErrorDuration, snackSuccessDuration } from '~/frontend/constants';

@Component({
  selector: 'app-dashboard',
  templateUrl: './dashboard.component.html',
  styleUrls: ['./dashboard.component.css'],
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [
    MatMenuModule,
    MatIconModule,
    MatButtonModule,
    MatChipsModule,
    MatTableModule,
    DatePipe,
    BytesPipe,
    ThumbnailIconComponent,
    DragDropDirective,
    DragPreviewDirective,
    DragItemDirective,
    DropTargetDirective,
    SelectionDirective,
    SelectionItemDirective,
    MatTooltipModule,
  ],
})
export class DashboardComponent {
  private readonly api = inject(Api);
  private readonly snackBar = inject(MatSnackBar);
  private readonly dialog = inject(MatDialog);
  private readonly clipboard = inject(Clipboard);
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
  protected readonly displayedColumns = [
    'icon',
    'name',
    'mimeType',
    'size',
    'createdAt',
    'updatedAt',
    'actions',
  ];
  protected readonly selection = viewChild.required<SelectionDirective<Datei>>(SelectionDirective);
  protected readonly uploading = signal(false);

  constructor() {
    effect(() => {
      this.dataSource.data = this.listDateiResource.value()?.items ?? [];
      this.selection().clear();
    });
  }

  protected onRowDblClick(row: Datei): void {
    if (row.isDirectory) {
      this.selection().clear();
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
      this.snackBar.open('Failed to load image', 'Dismiss', { duration: snackErrorDuration });
    }
  }

  private async downloadFile(id: string, name: string): Promise<void> {
    try {
      const response = await this.api.invoke$Response(downloadDatei, { id });
      triggerDownload(response.body as Blob, name);
    } catch (e) {
      console.error(e);
      this.snackBar.open('Failed to download file', 'Dismiss', { duration: snackErrorDuration });
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
        this.snackBar.open('Failed to create folder', 'Dismiss', { duration: snackErrorDuration });
      }
    });
  }

  protected openRenameDialog(datei: Datei): void {
    const ref = this.dialog.open(RenameDateiDialogComponent, {
      width: '360px',
      data: {
        currentName: datei.name ?? '',
        isDirectory: datei.isDirectory,
      } satisfies RenameDateiDialogData,
    });
    ref.afterClosed().subscribe(async (name: string | null) => {
      if (!name) return;
      try {
        await this.api.invoke(updateDatei$FormData, { id: datei.id, body: { name } });
        this.refresh.update((v) => v + 1);
      } catch (e) {
        console.error(e);
        this.snackBar.open('Failed to rename', 'Dismiss', { duration: snackErrorDuration });
      }
    });
  }

  protected async trashDatei(item: Datei, event: Event): Promise<void> {
    event.stopPropagation();
    await this.trash(item);
  }

  protected async trashSelected(): Promise<void> {
    await this.trash(this.selection().selected());
  }

  private async trash(items: Datei | Datei[]): Promise<void> {
    if (!Array.isArray(items)) {
      items = [items];
    }

    if (items.length === 0) {
      return;
    }

    const results = await Promise.allSettled(
      items.map((item) => this.api.invoke(deleteDatei, { id: item.id })),
    );

    const failed = results.filter((r) => r.status === 'rejected').length;
    if (failed > 0) {
      this.snackBar.open(`Failed to move ${failed} item(s) to trash`, 'Dismiss', {
        duration: snackErrorDuration,
      });
    }

    if (failed !== results.length) {
      this.refresh.update((v) => v + 1);
      const moved = items.length > 1 ? `${items.length - failed} items` : items[0].name || '1 item';
      this.snackBar.open(`Moved ${moved} to trash`, 'Dismiss', {
        duration: snackSuccessDuration,
      });
    }
  }

  protected createLinkForRow(row: Datei): void {
    this.openCreateLinkDialog([row.id], row.name ?? undefined);
  }

  protected createLinkForSelection(): void {
    const items = this.selection().selected();
    if (items.length === 0) return;
    const ids = items.map((i) => i.id);
    const defaultName = items.length === 1 ? (items[0].name ?? undefined) : undefined;
    this.openCreateLinkDialog(ids, defaultName);
  }

  private openCreateLinkDialog(dateiIds: string[], defaultName: string | undefined): void {
    const ref = this.dialog.open(LinkFormDialogComponent, {
      data: { mode: 'create', dateiIds, defaultName } satisfies LinkFormDialogData,
    });
    ref.afterClosed().subscribe((link) => {
      if (!link) return;
      const shareUrl = buildShareUrl(link.key);
      const snackRef = this.snackBar.open(`Public link "${link.name}" created`, 'Copy link', {
        duration: 6000,
      });
      snackRef.onAction().subscribe(() => {
        if (!this.clipboard.copy(shareUrl)) {
          this.snackBar.open('Failed to copy', 'Dismiss', { duration: snackErrorDuration });
        }
      });
      this.selection().clear();
    });
  }

  protected addToLinkForRow(row: Datei): void {
    this.openLinkPickerAndAdd([row.id]);
  }

  protected addToLinkForSelection(): void {
    const ids = this.selection()
      .selected()
      .map((d) => d.id);
    if (ids.length === 0) return;
    this.openLinkPickerAndAdd(ids);
  }

  private openLinkPickerAndAdd(dateiIds: string[]): void {
    const ref = this.dialog.open(LinkPickerDialogComponent);
    ref.afterClosed().subscribe(async (link: Link | undefined) => {
      if (!link) return;
      const results = await Promise.allSettled(
        dateiIds.map((dateiId) =>
          this.api.invoke(addDateiToLink, { id: link.id, body: { dateiId } }),
        ),
      );
      const failed = results.filter((r) => r.status === 'rejected').length;
      const added = results.length - failed;
      if (failed === 0) {
        this.snackBar.open(
          `Added ${added} ${added === 1 ? 'item' : 'items'} to "${link.name}"`,
          'OK',
          { duration: snackSuccessDuration },
        );
      } else {
        this.snackBar.open(
          `Added ${added} of ${results.length} items; ${failed} failed`,
          'Dismiss',
          { duration: snackErrorDuration },
        );
      }
      this.selection().clear();
    });
  }

  protected onDrag(event: DropEvent<Datei>): void {
    if (!event.target) return;
    if (!this.selection().isSelected(event.target)) {
      this.selection().setSelection(event.target);
    }
  }

  protected async onDrop(event: DropEvent<Datei>): Promise<void> {
    const items = this.selection()
      .selected()
      .filter((item) => item.id !== event.target?.id);
    const results = await Promise.allSettled(
      items.map((item) =>
        this.api.invoke(updateDatei$FormData, {
          id: item.id,
          body: { updateParentId: true, parentId: event.target?.id },
        }),
      ),
    );

    const failed = results.filter((r) => r.status === 'rejected').length;
    if (failed > 0) {
      this.snackBar.open(`Failed to move ${failed} item(s)`, 'Dismiss', {
        duration: snackErrorDuration,
      });
    }
    if (failed !== results.length) {
      this.refresh.update((v) => v + 1);
    }
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
