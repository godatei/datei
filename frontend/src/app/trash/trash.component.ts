import { JsonPipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, inject, resource } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { ActivatedRoute } from '@angular/router';
import { Api } from '~/api/api';
import { listTrash } from '~/api/functions';

@Component({
  selector: 'app-trash',
  templateUrl: './trash.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [JsonPipe],
})
export class TrashComponent {
  private readonly api = inject(Api);
  private readonly route = inject(ActivatedRoute);
  private readonly queryParams = toSignal(this.route.queryParamMap);
  protected readonly parentId = computed(() => this.queryParams()?.get('parentId') ?? null);

  protected readonly trashResource = resource({
    params: () => ({ parentId: this.parentId() }),
    loader: ({ params }) => this.api.invoke(listTrash, { parentId: params.parentId ?? undefined }),
  });
}
