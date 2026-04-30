import { Directive, ElementRef, inject, input } from '@angular/core';
import { DRAG_DROP } from './drag-drop.directive';

const INTERACTIVE_ELEMENT_SELECTOR =
  'button, a, input, select, textarea, label, summary, option, ' +
  '[role="button"], [role="link"], [contenteditable="true"], [tabindex], ' +
  '[matIconButton], [mat-button], [mat-raised-button], [mat-stroked-button], ' +
  '[mat-flat-button], [mat-icon-button], [mat-menu-item]';

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

  private isInteractiveTarget(target: EventTarget | null): boolean {
    return target instanceof Element && target.closest(INTERACTIVE_ELEMENT_SELECTOR) !== null;
  }

  protected onMouseDown(event: MouseEvent): void {
    if (this.isInteractiveTarget(event.target)) {
      return;
    }

    this.drag.startDrag(this.item(), event);
  }
}
