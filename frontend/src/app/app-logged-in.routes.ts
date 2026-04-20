import { Routes } from '@angular/router';
import { DashboardComponent } from '~/frontend/dashboard/dashboard.component';
import { UserSettingsComponent } from '~/frontend/settings/user-settings.component';

export const routes: Routes = [
  { path: '', pathMatch: 'full', component: DashboardComponent },
  { path: 'settings', component: UserSettingsComponent },
];
