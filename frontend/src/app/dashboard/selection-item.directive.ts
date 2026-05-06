import { computed, Directive, inject, input } from '@angular/core';
import { SelectionDirective } from './selection.directive';

@Directive({
  selector: '[appSelectionItem]',
  host: {
    '(click)': 'onHostClick($event)',
    '[class.selection-item-selected]': 'isSelected()',
  },
})
export class SelectionItemDirective<T extends { id: string }> {
  private readonly selection = inject<SelectionDirective<T>>(SelectionDirective);

  readonly item = input.required<T>({ alias: 'appSelectionItem' });

  protected readonly isSelected = computed(() => this.selection.selectedIds().has(this.item().id));

  protected onHostClick(event: MouseEvent): void {
    this.selection.handleClick(this.item(), event);
  }
}
