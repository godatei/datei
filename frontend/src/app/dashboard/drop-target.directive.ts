import { computed, Directive, ElementRef, inject, input } from '@angular/core';
import { DRAG_DROP } from './drag-drop.directive';

@Directive({
  selector: '[appDropTarget]',
  host: {
    '[class.drop-target]': 'dropTargetEnabled()',
    '[class.drop-target-active]': 'isActive()',
  },
})
export class DropTargetDirective<T> {
  private readonly drag = inject(DRAG_DROP);
  readonly nativeElement = inject<ElementRef<HTMLElement>>(ElementRef).nativeElement;

  readonly target = input.required<T | null>({ alias: 'appDropTarget' });
  readonly dropTargetEnabled = input(true);

  protected readonly isActive = computed(
    () =>
      this.dropTargetEnabled() &&
      this.drag.dragOverItem() !== undefined &&
      this.drag.dragOverItem() === this.target(),
  );
}
