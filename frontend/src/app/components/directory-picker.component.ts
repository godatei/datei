import {
  Component,
  computed,
  effect,
  inject,
  input,
  output,
  resource,
  signal,
  untracked,
} from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatListModule } from '@angular/material/list';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { Api } from '~/api/api';
import { listFiles } from '~/api/functions';
import { File, FilePathItem } from '~/api/models';

export interface DirectorySelection {
  id?: string;
  name: string;
}

@Component({
  selector: 'app-directory-picker',
  templateUrl: './directory-picker.component.html',
  styleUrl: './directory-picker.component.css',
  imports: [MatButtonModule, MatIconModule, MatListModule, MatProgressBarModule],
})
export class DirectoryPickerComponent {
  private readonly api = inject(Api);

  readonly initialPath = input<FilePathItem[]>([]);
  readonly currentChange = output<DirectorySelection | undefined>();

  protected readonly navItems = signal<FilePathItem[]>([]);
  protected readonly current = computed<DirectorySelection | undefined>(() => {
    const items = this.navItems();
    const last = items[items.length - 1];
    return last ? { id: last.id, name: last.name } : undefined;
  });

  protected readonly contents = resource({
    params: () => ({ parentId: this.current()?.id }),
    loader: async ({ params }) =>
      (await this.api.invoke(listFiles, params)).items.filter((it) => it.isDirectory),
  });

  constructor() {
    effect(() => {
      const path = this.initialPath();
      untracked(() => this.navItems.set(path));
    });
    effect(() => this.currentChange.emit(this.current()));
  }

  protected navigateTo(item: File): void {
    this.navItems.update((items) => items.concat({ id: item.id, name: item.name ?? '' }));
  }

  protected navigateUpTo(id?: string): void {
    this.navItems.update((items) => {
      if (id === undefined) {
        return [];
      }
      const i = items.findIndex((it) => it.id === id);
      return i >= 0 ? items.slice(0, i + 1) : [];
    });
  }
}
