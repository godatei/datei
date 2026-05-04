import { DatePipe } from '@angular/common';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  inject,
  resource,
  viewChild,
} from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatMenuModule } from '@angular/material/menu';
import { MatTableDataSource, MatTableModule } from '@angular/material/table';
import { ActivatedRoute, Router } from '@angular/router';
import { Api } from '~/api/api';
import { getDateiPath, listTrash } from '~/api/functions';
import { TrashedDatei } from '~/api/models';
import { ThumbnailIconComponent } from '~/frontend/dashboard/thumbnail-icon.component';
import { SelectionDirective } from '~/frontend/dashboard/selection.directive';
import { SelectionItemDirective } from '~/frontend/dashboard/selectable-item.directive';

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
  ],
})
export class TrashComponent {
  private readonly api = inject(Api);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly queryParams = toSignal(this.route.queryParamMap);
  protected readonly parentId = computed(() => this.queryParams()?.get('parentId') ?? null);

  protected readonly trashResource = resource({
    params: () => ({ parentId: this.parentId() }),
    loader: ({ params }) => this.api.invoke(listTrash, { parentId: params.parentId ?? undefined }),
  });

  protected readonly pathResource = resource({
    params: () => ({ parentId: this.parentId() }),
    loader: ({ params }) =>
      params.parentId
        ? this.api.invoke(getDateiPath, { id: params.parentId })
        : Promise.resolve([]),
  });

  protected readonly dataSource = new MatTableDataSource<TrashedDatei>([]);
  protected readonly displayedColumns = [
    'icon',
    'name',
    'trashedAt',
    'trashedBy',
    'originPath',
    'actions',
  ];
  protected readonly selection =
    viewChild.required<SelectionDirective<TrashedDatei>>(SelectionDirective);

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

  protected onRowDblClick(row: TrashedDatei): void {
    if (row.isDirectory) {
      this.selection().clear();
      this.router.navigate([], { relativeTo: this.route, queryParams: { parentId: row.id } });
    }
  }

  protected restore(item: TrashedDatei): void {
    // TODO: implement restore
    console.warn('restore not implemented', item);
  }

  protected deletePermanently(item: TrashedDatei): void {
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
