import { DatePipe, Location } from '@angular/common';
import { SelectionModel } from '@angular/cdk/collections';
import { Component, computed, effect, inject, resource, signal } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatDialog } from '@angular/material/dialog';
import { MatGridListModule } from '@angular/material/grid-list';
import { MatIconModule } from '@angular/material/icon';
import { MatMenuModule } from '@angular/material/menu';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatTableDataSource, MatTableModule } from '@angular/material/table';
import { ActivatedRoute, Router } from '@angular/router';
import { Api } from 'frontend/src/api/api';
import { createDatei, deleteDatei, listDatei } from 'frontend/src/api/functions';
import { Datei } from 'frontend/src/api/models';
import { NewFolderDialogComponent } from './new-folder-dialog.component';

@Component({
  selector: 'app-dashboard',
  templateUrl: './dashboard.component.html',
  styleUrls: ['./dashboard.component.css'],
  imports: [
    MatGridListModule,
    MatMenuModule,
    MatIconModule,
    MatButtonModule,
    MatCardModule,
    MatTableModule,
    MatProgressSpinnerModule,
    DatePipe,
  ],
})
export class DashboardComponent {
  private readonly api = inject(Api);
  private readonly snackBar = inject(MatSnackBar);
  private readonly dialog = inject(MatDialog);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly location = inject(Location);

  private readonly refresh = signal(0);
  private readonly queryParams = toSignal(this.route.queryParamMap);

  protected readonly parentId = computed(() => this.queryParams()?.get('parentId') ?? null);

  protected readonly listDateiResource = resource({
    params: () => ({ parentId: this.parentId(), refresh: this.refresh() }),
    loader: ({ params }) =>
      this.api.invoke(listDatei, params.parentId ? { parentId: params.parentId } : undefined),
  });
  protected readonly dataSource = new MatTableDataSource<Datei>([]);
  protected readonly displayedColumns = ['name', 'createdAt', 'updatedAt', 'mimeType', 'actions'];
  protected readonly selection = new SelectionModel<Datei>(true, [], true, (a, b) => a.id === b.id);
  protected readonly uploading = signal(false);

  constructor() {
    effect(() => {
      this.dataSource.data = this.listDateiResource.value()?.items ?? [];
      this.selection.clear();
    });
  }

  /** Based on the screen size, switch from standard to one column per row */
  cards = [
    { title: 'Card 1', cols: 1, rows: 1 },
    { title: 'Card 2', cols: 1, rows: 1 },
    { title: 'Card 3', cols: 1, rows: 1 },
    { title: 'Card 4', cols: 1, rows: 1 },
  ];

  protected onRowClick(row: Datei): void {
    this.selection.toggle(row);
  }

  protected onRowDblClick(row: Datei): void {
    if (!row.isDirectory) return;
    this.selection.clear();
    this.router.navigate([], { relativeTo: this.route, queryParams: { parentId: row.id } });
  }

  protected navigateUp(): void {
    this.location.back();
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
