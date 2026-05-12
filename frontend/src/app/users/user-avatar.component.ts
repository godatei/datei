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
  readonly sizeClasses = computed(() => {
    switch (this.size()) {
      case 'sm':
        return 'w-8 h-8 mat-font-title-sm';
      case 'lg':
        return 'w-16 h-16 mat-font-headline-sm';
      default:
        return 'w-12 h-12 mat-font-title-md';
    }
  });
}
