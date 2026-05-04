import {
  contentChildren,
  DestroyRef,
  Directive,
  forwardRef,
  inject,
  InjectionToken,
  output,
  signal,
} from '@angular/core';
import { DropTargetDirective } from './drop-target.directive';

export interface DropEvent<T> {
  target: T | null;
}

export const DRAG_DROP = new InjectionToken<DragDropDirective<unknown>>('DragDropDirective');

@Directive({
  selector: '[appDragDrop]',
  exportAs: 'appDragDrop',
  providers: [{ provide: DRAG_DROP, useExisting: forwardRef(() => DragDropDirective) }],
  host: { '[class.cursor-grabbing]': 'isDragging()' },
})
export class DragDropDirective<T> {
  private readonly destroyRef = inject(DestroyRef);
  private readonly dropTargets = contentChildren<DropTargetDirective<T>>(DropTargetDirective, {
    descendants: true,
  });

  readonly dragStart = output<DropEvent<T>>();
  readonly dropped = output<DropEvent<T>>();

  readonly isDragging = signal(false);
  readonly dragOverItem = signal<T | null | undefined>(undefined);
  readonly dragPointerPos = signal({ x: 0, y: 0 });

  private dragStartPos: { x: number; y: number } | null = null;
  private dragStartItem: T | null = null;

  private readonly onPointerMoveBound = this.onPointerMove.bind(this);
  private readonly onPointerUpBound = this.onPointerUp.bind(this);

  constructor() {
    this.destroyRef.onDestroy(() => {
      document.removeEventListener('mousemove', this.onPointerMoveBound);
      document.removeEventListener('mouseup', this.onPointerUpBound);
    });
  }

  startDrag(item: T, event: MouseEvent): void {
    if (event.button !== 0) return;
    this.dragStartPos = { x: event.clientX, y: event.clientY };
    this.dragStartItem = item;
    document.addEventListener('mousemove', this.onPointerMoveBound);
    document.addEventListener('mouseup', this.onPointerUpBound);
  }

  private onPointerMove(event: MouseEvent): void {
    if (!this.dragStartPos || !this.dragStartItem) return;

    if (!this.isDragging()) {
      const dx = event.clientX - this.dragStartPos.x;
      const dy = event.clientY - this.dragStartPos.y;
      if (Math.hypot(dx, dy) < 5) {
        return;
      }

      this.dragStart.emit({ target: this.dragStartItem });
      this.isDragging.set(true);
    }

    this.dragPointerPos.set({ x: event.clientX, y: event.clientY });

    const el = document.elementFromPoint(event.clientX, event.clientY);
    const target = this.findDropTarget(el);
    this.dragOverItem.set(target);
  }

  private findDropTarget(el: Element | null): T | null | undefined {
    let node: Element | null = el;
    while (node) {
      const match = this.dropTargets().find((t) => t.nativeElement === node);
      if (match) {
        return match.dropTargetEnabled() ? match.target() : undefined;
      }
      node = node.parentElement;
    }
    return undefined;
  }

  private onPointerUp(): void {
    document.removeEventListener('mousemove', this.onPointerMoveBound);
    document.removeEventListener('mouseup', this.onPointerUpBound);

    const target = this.dragOverItem();
    const wasDragging = this.isDragging();

    this.isDragging.set(false);
    this.dragOverItem.set(undefined);
    this.dragStartPos = null;
    this.dragStartItem = null;

    if (wasDragging) {
      // Suppress the synthetic click the browser fires after mouseup on the same element.
      document.addEventListener('click', (e) => e.stopPropagation(), { capture: true, once: true });
    }

    if (!wasDragging || target === undefined) {
      return;
    }

    this.dropped.emit({ target });
  }
}
