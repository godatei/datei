import { DatePipe } from '@angular/common';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  inject,
  resource,
  signal,
  viewChild,
} from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { MatButtonModule } from '@angular/material/button';
import { MatDialog } from '@angular/material/dialog';
import { MatIconModule } from '@angular/material/icon';
import { MatMenuModule } from '@angular/material/menu';
import { MatTableDataSource, MatTableModule } from '@angular/material/table';
import { ActivatedRoute, Router } from '@angular/router';
import { Api } from '~/api/api';
import { getDateiPath, listTrash, listTrashChildren } from '~/api/functions';
import { Datei, TrashedDatei } from '~/api/models';
import { ThumbnailIconComponent } from '~/frontend/dashboard/thumbnail-icon.component';
import { SelectionDirective } from '~/frontend/dashboard/selection.directive';
import { SelectionItemDirective } from '~/frontend/dashboard/selection-item.directive';
import { RestoreDialogComponent } from './restore-dialog/restore-dialog.component';
import { filter } from 'rxjs';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';

@Component({
  selector: 'app-trash',
  templateUrl: './trash.component.html',
  styleUrls: ['./trash.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    DatePipe,
    MatButtonModule,
    MatIconModule,
    MatMenuModule,
    MatTableModule,
    ThumbnailIconComponent,
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
        ? this.api.invoke(listTrashChildren, { dateiId: params.parentId })
        : this.api.invoke(listTrash, undefined),
  });

  protected readonly pathResource = resource({
    params: () => ({ parentId: this.parentId() }),
    loader: ({ params }) =>
      params.parentId
        ? this.api.invoke(getDateiPath, { id: params.parentId })
        : Promise.resolve([]),
  });

  protected readonly displayedColumns = computed(() =>
    this.parentId()
      ? ['icon', 'name', 'actions']
      : ['icon', 'name', 'trashedAt', 'originPath', 'actions'],
  );

  protected readonly dataSource = new MatTableDataSource<Datei>([]);
  protected readonly selection = viewChild.required<SelectionDirective<Datei>>(SelectionDirective);

  constructor() {
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

  protected onRowDblClick(row: Datei): void {
    if (row.isDirectory) {
      this.selection().clear();
      this.router.navigate([], { relativeTo: this.route, queryParams: { parentId: row.id } });
    }
  }

  protected restore(item: Datei): void {
    const dialogRef = this.dialog.open(RestoreDialogComponent, { data: item });
    dialogRef
      .afterClosed()
      .pipe(filter((result) => result))
      .subscribe((result: { parent?: Datei }) => {
        this.refresh.update((v) => v + 1);
        console.log(result);
        const snackRef = this.snack.open(
          `"${item.name}" has been restored to ${result.parent?.name ?? 'My Files'}`,
          'Open location',
        );
        snackRef
          .onAction()
          .subscribe(() =>
            this.router.navigate(['/'], { queryParams: { parentId: result.parent?.id ?? null } }),
          );
      });
  }

  protected deletePermanently(item: Datei): void {
    // TODO: implement permanent delete
    console.warn('delete not implemented', item);
  }

  protected formatOriginPath(item: TrashedDatei): string {
    const parts = item.originPath;
    if (!parts || parts.length === 0) {
      return 'My files';
    }
    return ['My files', ...parts.map((p) => p.name)].join(' / ');
  }
}
