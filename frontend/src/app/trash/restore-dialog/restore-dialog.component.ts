import { Component, computed, inject, OnInit, resource, signal } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatIconModule } from '@angular/material/icon';
import { MatListModule } from '@angular/material/list';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatSnackBar } from '@angular/material/snack-bar';
import { Api } from '~/api/api';
import { snackErrorDuration } from '~/frontend/constants';
import { getDateiPath, listDatei, restoreTrash } from '~/api/functions';
import { Datei, DateiPathItem } from '~/api/models';

@Component({
  templateUrl: './restore-dialog.component.html',
  styleUrl: './restore-dialog.component.css',
  imports: [MatButtonModule, MatDialogModule, MatIconModule, MatListModule, MatProgressBarModule],
})
export class RestoreDialogComponent implements OnInit {
  protected readonly data = inject<Datei>(MAT_DIALOG_DATA);
  protected readonly dialogRef = inject(MatDialogRef<RestoreDialogComponent>);
  private readonly api = inject(Api);
  private readonly snack = inject(MatSnackBar);

  protected readonly navItems = signal<DateiPathItem[]>([]);
  protected readonly currentNavItem = computed(() => {
    const items = this.navItems();
    return items.length > 0 ? items[items.length - 1] : undefined;
  });

  protected readonly currentContents = resource({
    params: () => {
      const items = this.navItems();
      return { parentId: items.length > 0 ? items[items.length - 1]?.id : undefined };
    },
    loader: async ({ params }) =>
      (await this.api.invoke(listDatei, params)).items.filter((it) => it.isDirectory),
  });

  public async ngOnInit(): Promise<void> {
    if (this.data.parentId) {
      try {
        // initialize the directory picker with the original parent unless it is also trashed
        const path = await this.api.invoke(getDateiPath, { id: this.data.parentId });
        if (!path.some((it) => it.trashed)) {
          this.navItems.set(path);
        }
      } catch (e) {
        console.error(e);
        this.snack.open('Failed to load original path', 'Dismiss', {
          duration: snackErrorDuration,
        });
      }
    }
  }

  protected navigateTo(item: Datei) {
    this.navItems.update((items) => items.concat({ id: item.id, name: item.name ?? '' }));
  }

  protected navigateUpTo(id?: string) {
    this.navItems.update((items) => {
      if (id === undefined) {
        return [];
      }
      const i = items.findIndex((it) => it.id === id);
      return i >= 0 ? items.slice(0, i + 1) : [];
    });
  }

  protected async restore() {
    try {
      const parent = this.currentNavItem();
      await this.api.invoke(restoreTrash, {
        dateiId: this.data.id,
        body: { parentId: parent?.id ?? null },
      });
      this.dialogRef.close({ parent });
    } catch (e) {
      console.error(e);
      this.snack.open(
        `Failed to restore ${this.data.name ?? 'Unnamed'} in ${parent?.name ?? 'My files'}`,
        'Dismiss',
        {
          duration: snackErrorDuration,
        },
      );
    }
  }
}
