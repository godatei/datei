import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { initials } from './initials';

@Component({
  selector: 'app-user-avatar',
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './user-avatar.component.html',
})
export class UserAvatarComponent {
  readonly name = input.required<string | null | undefined>();
  readonly size = input<'sm' | 'md' | 'lg'>('md');

  readonly letters = computed(() => initials(this.name()));
  readonly classes = computed(() => {
    const base =
      'inline-flex items-center justify-center mat-corner-full mat-bg-primary-container mat-text-on-primary-container shrink-0 select-none align-middle';
    switch (this.size()) {
      case 'sm':
        return `${base} w-8 h-8 mat-font-title-sm`;
      case 'lg':
        return `${base} w-16 h-16 mat-font-headline-sm`;
      default:
        return `${base} w-12 h-12 mat-font-title-md`;
    }
  });
}
