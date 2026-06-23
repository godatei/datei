import { Component, computed, inject, signal } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatIconModule } from '@angular/material/icon';
import { MatListModule } from '@angular/material/list';
import { Api } from '~/api/api';
import { listFiles } from '~/api/functions';
import type { File as ApiFile } from '~/api/models/file';

export interface DirectorySelection {
  id?: string;
  name: string;
}

interface Crumb {
  id?: string;
  name: string;
}

@Component({
  selector: 'app-directory-picker-dialog',
  templateUrl: './directory-picker-dialog.component.html',
  imports: [MatDialogModule, MatButtonModule, MatIconModule, MatListModule],
})
export class DirectoryPickerDialogComponent {
  private readonly api = inject(Api);
  private readonly dialogRef = inject(
    MatDialogRef<DirectoryPickerDialogComponent, DirectorySelection | undefined>,
  );

  protected readonly path = signal<Crumb[]>([{ name: 'Root' }]);
  protected readonly directories = signal<ApiFile[]>([]);
  protected readonly loading = signal(false);
  protected readonly error = signal(false);

  protected readonly current = computed(() => this.path()[this.path().length - 1]);

  constructor() {
    void this.load();
  }

  private async load(): Promise<void> {
    this.loading.set(true);
    this.error.set(false);
    try {
      const res = await this.api.invoke(listFiles, { parentId: this.current().id, limit: 500 });
      this.directories.set(res.items.filter((f) => f.isDirectory));
    } catch (e) {
      console.error(e);
      this.error.set(true);
    } finally {
      this.loading.set(false);
    }
  }

  protected enter(dir: ApiFile): void {
    this.path.update((p) => [...p, { id: dir.id, name: dir.name ?? 'Untitled' }]);
    void this.load();
  }

  protected goTo(index: number): void {
    this.path.update((p) => p.slice(0, index + 1));
    void this.load();
  }

  protected selectCurrent(): void {
    const crumb = this.current();
    this.dialogRef.close({ id: crumb.id, name: crumb.name });
  }

  protected cancel(): void {
    this.dialogRef.close(undefined);
  }
}
