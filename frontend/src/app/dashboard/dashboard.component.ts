import { DatePipe } from '@angular/common';
import { Component, effect, inject, resource, signal } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatGridListModule } from '@angular/material/grid-list';
import { MatIconModule } from '@angular/material/icon';
import { MatMenuModule } from '@angular/material/menu';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatSnackBar } from '@angular/material/snack-bar';
import { MatTableDataSource, MatTableModule } from '@angular/material/table';
import { Api } from 'frontend/src/api/api';
import { createDatei, listDatei } from 'frontend/src/api/functions';
import { DateiResponse } from 'frontend/src/api/models';

@Component({
  selector: 'app-dashboard',
  templateUrl: './dashboard.component.html',
  styleUrl: './dashboard.component.css',
  imports: [
    MatGridListModule,
    MatMenuModule,
    MatIconModule,
    MatButtonModule,
    MatCardModule,
    MatTableModule,
    MatProgressSpinnerModule,
    DatePipe,
  ],
})
export class DashboardComponent {
  private readonly api = inject(Api);
  private readonly snackBar = inject(MatSnackBar);

  private readonly refresh = signal(0);

  protected readonly sres = resource({
    params: () => ({ refresh: this.refresh() }),
    loader: () => this.api.invoke(listDatei),
  });
  protected readonly dataSource = new MatTableDataSource<DateiResponse>([]);
  protected readonly displayedColumns = ['name', 'createdAt', 'updatedAt', 'mimeType'];
  protected readonly uploading = signal(false);

  constructor() {
    effect(() => (this.dataSource.data = this.sres.value()?.items ?? []));
  }

  /** Based on the screen size, switch from standard to one column per row */
  cards = [
    { title: 'Card 1', cols: 1, rows: 1 },
    { title: 'Card 2', cols: 1, rows: 1 },
    { title: 'Card 3', cols: 1, rows: 1 },
    { title: 'Card 4', cols: 1, rows: 1 },
  ];

  protected async startUpload(el: HTMLInputElement) {
    if (el.files === null || el.files.length === 0) {
      return;
    }

    const snack = this.snackBar.open('Upload in progres…');
    this.uploading.set(true);
    try {
      const file = el.files[0];
      await this.api.invoke(createDatei, { body: { name: file.name, file: file } });
      this.refresh.update((v) => v + 1);
    } catch (e) {
      console.error(e);
    } finally {
      setTimeout(() => snack.dismiss(), 500);
      this.uploading.set(false);
    }
  }
}
