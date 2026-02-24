import { Component, signal } from '@angular/core';
import { NavComponent } from '~/frontend/nav/nav.component';

@Component({
  selector: 'app-root',
  imports: [NavComponent],
  templateUrl: './app.html',
  styleUrl: './app.css',
})
export class App {
  protected readonly title = signal('Datei');
}
