import { Component, computed, inject } from '@angular/core';
import { AuthService } from '~/frontend/services/auth.service';
import { UserAvatarComponent } from '~/frontend/users/user-avatar/user-avatar.component';
import { ownerLabel } from '~/util/owner';

/** Table cell showing the file's owner. Ownership is single-user, so it always
 *  shows the current user as an initials avatar plus a "me" label. */
@Component({
  selector: 'app-owner-cell',
  template: `
    <div class="flex items-center gap-2">
      <app-user-avatar [name]="avatarName()" size="2xs" />
      <span>{{ label }}</span>
    </div>
  `,
  imports: [UserAvatarComponent],
})
export class OwnerCellComponent {
  private readonly auth = inject(AuthService);

  protected readonly label = ownerLabel();
  protected readonly avatarName = computed(() => this.auth.userName() ?? '');
}
