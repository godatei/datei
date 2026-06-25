import { Component, computed, effect, inject, resource, signal, viewChild } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { MatButtonModule } from '@angular/material/button';
import { MatChipsModule } from '@angular/material/chips';
import { MatDialog } from '@angular/material/dialog';
import { MatIconModule } from '@angular/material/icon';
import { MatMenuModule } from '@angular/material/menu';
import { MatSort, MatSortModule } from '@angular/material/sort';
import { MatTableDataSource, MatTableModule } from '@angular/material/table';
import { ActivatedRoute, Router } from '@angular/router';
import { Api } from '~/api/api';
import { getFilePath, listTrash, listTrashChildren } from '~/api/functions';
import { File, TrashedFile } from '~/api/models';
import { FileIconComponent } from '~/frontend/dashboard/file-icon.component';
import { SelectionDirective } from '~/frontend/dashboard/selection.directive';
import { SelectionItemDirective } from '~/frontend/dashboard/selection-item.directive';
import { OwnerCellComponent } from '~/frontend/shared/owner-cell.component';
import { ownerLabel } from '~/util/owner';
import { RestoreDialogComponent } from './restore-dialog/restore-dialog.component';
import { SmartDatePipe } from '~/frontend/pipes/smart-date.pipe';
import { filter } from 'rxjs';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { snackSuccessDuration } from '~/frontend/constants';

@Component({
  selector: 'app-trash',
  templateUrl: './trash.component.html',
  host: { class: 'flex flex-col grow min-h-0' },
  imports: [
    SmartDatePipe,
    MatButtonModule,
    MatChipsModule,
    MatIconModule,
    MatMenuModule,
    MatTableModule,
    MatSortModule,
    FileIconComponent,
    OwnerCellComponent,
    SelectionDirective,
    SelectionItemDirective,
    MatSnackBarModule,
  ],
})
export class TrashComponent {
  private readonly api = inject(Api);
  private readonly dialog = inject(MatDialog);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly snack = inject(MatSnackBar);

  private readonly queryParams = toSignal(this.route.queryParamMap);
  protected readonly parentId = computed(() => this.queryParams()?.get('parentId') ?? null);

  protected readonly refresh = signal(0);

  protected readonly trashResource = resource({
    params: () => ({ refresh: this.refresh(), parentId: this.parentId() }),
    loader: ({ params }) =>
      params.parentId
        ? this.api.invoke(listTrashChildren, { fileId: params.parentId })
        : this.api.invoke(listTrash, undefined),
  });

  protected readonly pathResource = resource({
    params: () => ({ parentId: this.parentId() }),
    loader: ({ params }) =>
      params.parentId ? this.api.invoke(getFilePath, { id: params.parentId }) : Promise.resolve([]),
  });

  protected readonly displayedColumns = computed(() =>
    this.parentId()
      ? ['name', 'owner', 'actions']
      : ['name', 'owner', 'trashedAt', 'originPath', 'actions'],
  );

  protected readonly dataSource = new MatTableDataSource<File>([]);
  protected readonly selection = viewChild.required<SelectionDirective<File>>(SelectionDirective);
  protected readonly sort = viewChild(MatSort);

  constructor() {
    this.dataSource.sortingDataAccessor = (file, column) => {
      switch (column) {
        case 'name':
          return file.name?.toLowerCase() ?? '';
        case 'owner':
          return ownerLabel();
        case 'trashedAt':
          return file.trashedAt ? new Date(file.trashedAt).getTime() : 0;
        case 'originPath':
          return this.formatOriginPath(file as TrashedFile).toLowerCase();
        default:
          return '';
      }
    };

    effect(() => {
      const sort = this.sort();
      if (sort) this.dataSource.sort = sort;
    });

    // Navigating into a folder drops the trashedAt/originPath columns. Clear any
    // active sort on a now-hidden column so rows aren't left ordered by an
    // invisible column with no header indicator.
    effect(() => {
      const columns = this.displayedColumns();
      const sort = this.sort();
      if (sort && sort.active && !columns.includes(sort.active)) {
        sort.active = '';
        sort.direction = '';
        sort.sortChange.emit({ active: '', direction: '' });
      }
    });

    effect(() => {
      this.dataSource.data = this.trashResource.value()?.items ?? [];
      this.selection().clear();
    });
  }

  protected navigateTo(id: string | null): void {
    this.router.navigate([], {
      relativeTo: this.route,
      queryParams: id ? { parentId: id } : {},
    });
  }

  protected onRowDblClick(row: File): void {
    if (row.isDirectory) {
      this.selection().clear();
      this.router.navigate([], { relativeTo: this.route, queryParams: { parentId: row.id } });
    }
  }

  protected restore(item: File): void {
    const dialogRef = this.dialog.open(RestoreDialogComponent, { data: item });
    dialogRef
      .afterClosed()
      .pipe(filter((result) => result))
      .subscribe((result: { parent?: File }) => {
        this.refresh.update((v) => v + 1);
        const snackRef = this.snack.open(
          `"${item.name ?? 'Unnamed'}" has been restored to ${result.parent?.name ?? 'My files'}`,
          'Open location',
          { duration: snackSuccessDuration },
        );
        snackRef
          .onAction()
          .subscribe(() =>
            this.router.navigate(['/'], { queryParams: { parentId: result.parent?.id ?? null } }),
          );
      });
  }

  protected deletePermanently(item: File): void {
    // TODO: implement permanent delete
    console.warn('delete not implemented', item);
  }

  protected formatOriginPath(item: TrashedFile): string {
    const parts = item.originPath;
    if (!parts || parts.length === 0) {
      return 'My files';
    }
    return ['My files', ...parts.map((p) => p.name)].join(' / ');
  }
}
