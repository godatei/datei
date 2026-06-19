import { Component } from '@angular/core';
import { MatListModule } from '@angular/material/list';
import { RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';

@Component({
  selector: 'app-user-settings',
  imports: [MatListModule, RouterLink, RouterLinkActive, RouterOutlet],
  templateUrl: './user-settings.component.html',
  host: { class: 'block h-full' },
})
export class UserSettingsComponent {}
