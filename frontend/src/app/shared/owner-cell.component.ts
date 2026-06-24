import { Component, computed, inject, input } from '@angular/core';
import { File } from '~/api/models';
import { AuthService } from '~/frontend/services/auth.service';
import { UserAvatarComponent } from '~/frontend/users/user-avatar/user-avatar.component';
import { ownerLabel } from '~/util/owner';

/** Table cell showing a file's owner as an initials avatar plus a label. */
@Component({
  selector: 'app-owner-cell',
  template: `
    <div class="flex items-center gap-2">
      <app-user-avatar [name]="avatarName()" size="2xs" />
      <span>{{ label() }}</span>
    </div>
  `,
  imports: [UserAvatarComponent],
})
export class OwnerCellComponent {
  public readonly file = input.required<File>();

  private readonly auth = inject(AuthService);
  private readonly currentUserId = computed(() => this.auth.getClaims()?.sub);
  private readonly isMe = computed(() => {
    const ownerId = this.file().ownerId;
    return !ownerId || ownerId === this.currentUserId();
  });

  protected readonly label = computed(() => ownerLabel(this.file(), this.currentUserId()));
  protected readonly avatarName = computed(() =>
    this.isMe() ? (this.auth.userName() ?? '') : (this.file().ownerName ?? ''),
  );
}
