import { Routes } from '@angular/router';
import { DashboardComponent } from '~/frontend/dashboard/dashboard.component';

export const routes: Routes = [
  {
    path: '',
    pathMatch: 'full',
    component: DashboardComponent,
  },
];
