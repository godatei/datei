import { Directive, ElementRef, inject, input } from '@angular/core';
import { DRAG_DROP } from './drag-drop.directive';

@Directive({
  selector: '[appDragItem]',
  host: {
    '(mousedown)': 'onMouseDown($event)',
  },
})
export class DragItemDirective<T> {
  protected readonly drag = inject(DRAG_DROP);
  readonly nativeElement = inject<ElementRef<HTMLElement>>(ElementRef).nativeElement;

  readonly item = input.required<T>({ alias: 'appDragItem' });

  protected onMouseDown(event: MouseEvent): void {
    this.drag.startDrag(this.item(), event);
  }
}
