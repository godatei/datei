import { Routes } from '@angular/router';
import { DashboardComponent } from '~/frontend/dashboard/dashboard.component';
import { UserSettingsComponent } from '~/frontend/settings/user-settings.component';
import { TrashComponent } from '~/frontend/trash/trash.component';

export const routes: Routes = [
  { path: '', pathMatch: 'full', component: DashboardComponent },
  { path: 'trash', component: TrashComponent },
  { path: 'settings', component: UserSettingsComponent },
  {
    path: 'links',
    loadComponent: () =>
      import('~/frontend/links/links-list/links-list.component').then((m) => m.LinksListComponent),
  },
];
