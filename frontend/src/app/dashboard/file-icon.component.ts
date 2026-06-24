import { Component, computed, input } from '@angular/core';
import { MatIconModule } from '@angular/material/icon';
import { File } from '~/api/models';
import {
  FILE_ICON_FALLBACK,
  FOLDER_ICON,
  LSICON_NAMESPACE,
  fileIconColor,
  lsiconForFile,
} from '~/util/file-icons';

@Component({
  selector: 'app-file-icon',
  host: { class: 'flex items-center justify-center' },
  template: `
    @if (svgIcon(); as icon) {
      <mat-icon [svgIcon]="icon" [style.color]="color()" />
    } @else {
      <mat-icon [style.color]="color()">{{ fallbackIcon() }}</mat-icon>
    }
  `,
  styles: `
    :host {
      /* Smaller than the default 24px icon to suit the compact files table. */
      --datei-file-icon-size: 1.25rem;
    }
    mat-icon {
      width: var(--datei-file-icon-size);
      height: var(--datei-file-icon-size);
      font-size: var(--datei-file-icon-size);
      line-height: var(--datei-file-icon-size);
    }
  `,
  imports: [MatIconModule],
})
export class FileIconComponent {
  public readonly file = input.required<File>();

  protected readonly svgIcon = computed(() => {
    const name = lsiconForFile(this.file());
    return name ? `${LSICON_NAMESPACE}:${name}` : null;
  });

  protected readonly fallbackIcon = computed(() =>
    this.file().isDirectory ? FOLDER_ICON : FILE_ICON_FALLBACK,
  );

  protected readonly color = computed(() => fileIconColor(this.file()));
}
