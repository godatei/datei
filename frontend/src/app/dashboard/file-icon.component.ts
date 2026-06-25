import { Component, computed, input } from '@angular/core';
import { MatIconModule } from '@angular/material/icon';
import { File } from '~/api/models';
import {
  FILE_ICON_FALLBACK,
  FOLDER_ICON,
  LSICON_NAMESPACE,
  fileIconColorClass,
  lsiconForFile,
} from '~/util/file-icons';

@Component({
  selector: 'app-file-icon',
  host: { class: 'flex items-center justify-center' },
  template: `
    @if (svgIcon(); as icon) {
      <mat-icon [svgIcon]="icon" [class]="colorClass()" />
    } @else {
      <mat-icon [class]="colorClass()">{{ fallbackIcon() }}</mat-icon>
    }
  `,
  styles: `
    @reference 'tailwindcss';

    mat-icon {
      /* Smaller than the default 24px for the compact files table. size-5 covers
         width/height (used by SVG icons); font-size/line-height size the ligature
         fallback used for folders and unknown file types. */
      @apply size-5;
      font-size: 1.25rem;
      line-height: 1.25rem;
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

  protected readonly colorClass = computed(() => fileIconColorClass(this.file()));
}
