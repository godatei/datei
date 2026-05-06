import { computed, Directive, input, signal } from '@angular/core';

@Directive({
  selector: '[appSelection]',
  exportAs: 'appSelection',
})
export class SelectionDirective<T extends { id: string }> {
  readonly items = input<T[]>([], { alias: 'appSelection' });

  private readonly _selectedIds = signal<ReadonlySet<string>>(new Set());
  private anchor: T | null = null;

  readonly selectedIds = this._selectedIds.asReadonly();
  readonly selected = computed(() =>
    this.items().filter((item) => this._selectedIds().has(item.id)),
  );
  readonly size = computed(() => this.selectedIds().size);
  readonly isAnySelected = computed(() => this.size() > 0);

  handleClick(item: T, event: MouseEvent): void {
    if (event.shiftKey && this.anchor !== null) {
      const data = this.items();
      const anchorIdx = data.findIndex((d) => d.id === this.anchor!.id);
      const rowIdx = data.findIndex((d) => d.id === item.id);
      if (anchorIdx !== -1 && rowIdx !== -1) {
        const [lo, hi] = anchorIdx <= rowIdx ? [anchorIdx, rowIdx] : [rowIdx, anchorIdx];
        this._selectedIds.set(new Set(data.slice(lo, hi + 1).map((d) => d.id)));
      }
    } else if (event.ctrlKey || event.metaKey) {
      this._selectedIds.update((ids) => {
        const next = new Set(ids);
        if (next.has(item.id)) next.delete(item.id);
        else next.add(item.id);
        return next;
      });
      this.anchor = item;
    } else {
      this._selectedIds.set(new Set([item.id]));
      this.anchor = item;
    }
  }

  clear(): void {
    this._selectedIds.set(new Set());
    this.anchor = null;
  }

  isSelected(item: T): boolean {
    return this._selectedIds().has(item.id);
  }

  setSelection(item: T): void {
    this._selectedIds.set(new Set([item.id]));
  }
}
