import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { SafeUrl } from '@angular/platform-browser';

// Presentational image preview. Consumers handle blob loading and object-URL
// lifecycle; this component just renders the <img>. Used by the dashboard's
// preview dialog and the public link viewer's single-file landing UI.
@Component({
  selector: 'app-image-preview',
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<img [src]="src()" [alt]="alt()" class="max-w-full block h-auto" />`,
})
export class ImagePreviewComponent {
  readonly src = input.required<SafeUrl>();
  readonly alt = input<string>('');
}
