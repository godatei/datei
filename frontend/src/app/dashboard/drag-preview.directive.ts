import { computed, Directive, inject } from '@angular/core';
import { DRAG_DROP } from './drag-drop.directive';

@Directive({
  selector: '[appDragPreview]',
  host: {
    '[class.hidden]': '!drag.isDragging()',
    '[style.left.px]': 'left()',
    '[style.top.px]': 'top()',
  },
})
export class DragPreviewDirective {
  protected readonly drag = inject(DRAG_DROP);

  protected readonly left = computed(() => this.drag.dragPointerPos().x + 16);
  protected readonly top = computed(() => this.drag.dragPointerPos().y + 16);
}
