import {
  DestroyRef,
  Directive,
  forwardRef,
  inject,
  input,
  InjectionToken,
  output,
  signal,
} from '@angular/core';
import { SelectionModel } from '@angular/cdk/collections';
import { Datei } from 'frontend/src/api/models';

export interface DropEvent {
  items: Datei[];
  targetId: string;
}

export const DRAG_DROP = new InjectionToken<DragDropDirective>('DragDropDirective');

@Directive({
  selector: '[appDragDrop]',
  exportAs: 'appDragDrop',
  providers: [{ provide: DRAG_DROP, useExisting: forwardRef(() => DragDropDirective) }],
  host: { '[class.cursor-grabbing]': 'isDragging()' },
})
export class DragDropDirective {
  private readonly destroyRef = inject(DestroyRef);

  readonly dragSelection = input.required<SelectionModel<Datei>>();
  readonly dropped = output<DropEvent>();

  readonly isDragging = signal(false);
  readonly dragOverDirectoryId = signal<string | null>(null);
  readonly dragPointerPos = signal({ x: 0, y: 0 });
  readonly dragItems = signal<Datei[]>([]);

  private dragStartPos: { x: number; y: number } | null = null;
  private dragStartRow: Datei | null = null;

  private readonly onPointerMoveBound = this.onPointerMove.bind(this);
  private readonly onPointerUpBound = this.onPointerUp.bind(this);

  constructor() {
    this.destroyRef.onDestroy(() => {
      document.removeEventListener('mousemove', this.onPointerMoveBound);
      document.removeEventListener('mouseup', this.onPointerUpBound);
    });
  }

  onRowMouseDown(row: Datei, event: MouseEvent): void {
    if (event.button !== 0) return;
    this.dragStartPos = { x: event.clientX, y: event.clientY };
    this.dragStartRow = row;
    document.addEventListener('mousemove', this.onPointerMoveBound);
    document.addEventListener('mouseup', this.onPointerUpBound);
  }

  private onPointerMove(event: MouseEvent): void {
    if (!this.dragStartPos || !this.dragStartRow) return;

    if (!this.isDragging()) {
      const dx = event.clientX - this.dragStartPos.x;
      const dy = event.clientY - this.dragStartPos.y;
      if (Math.hypot(dx, dy) < 5) return;

      const selection = this.dragSelection();
      if (!selection.isSelected(this.dragStartRow)) {
        selection.clear();
        selection.select(this.dragStartRow);
      }
      this.dragItems.set([...selection.selected]);
      this.isDragging.set(true);
    }

    this.dragPointerPos.set({ x: event.clientX, y: event.clientY });

    const el = document.elementFromPoint(event.clientX, event.clientY);
    const target = el?.closest<HTMLElement>('[data-drop-target]');
    if (target) {
      const id = target.dataset['dropTarget']!;
      if (!this.dragItems().some((d) => d.id === id)) {
        this.dragOverDirectoryId.set(id);
        return;
      }
    }
    this.dragOverDirectoryId.set(null);
  }

  private onPointerUp(): void {
    document.removeEventListener('mousemove', this.onPointerMoveBound);
    document.removeEventListener('mouseup', this.onPointerUpBound);

    const targetId = this.dragOverDirectoryId();
    const wasDragging = this.isDragging();

    this.isDragging.set(false);
    this.dragOverDirectoryId.set(null);
    this.dragStartPos = null;
    this.dragStartRow = null;

    if (wasDragging) {
      // Suppress the synthetic click the browser fires after mouseup on the same element.
      document.addEventListener('click', (e) => e.stopPropagation(), { capture: true, once: true });
    }

    if (!wasDragging || targetId === null) return;

    this.dropped.emit({ items: this.dragItems(), targetId });
  }
}
