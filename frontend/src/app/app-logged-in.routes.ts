import { Routes } from '@angular/router';
import { AdminUserDetailComponent } from '~/frontend/admin/admin-user-detail.component';
import { AdminUsersListComponent } from '~/frontend/admin/admin-users-list.component';
import { DashboardComponent } from '~/frontend/dashboard/dashboard.component';
import { adminGuard } from '~/frontend/guards/admin.guard';
import { LinksListComponent } from '~/frontend/links/links-list/links-list.component';
import { UserSettingsComponent } from '~/frontend/settings/user-settings.component';
import { TrashComponent } from '~/frontend/trash/trash.component';

export const routes: Routes = [
  { path: '', pathMatch: 'full', component: DashboardComponent },
  { path: 'trash', component: TrashComponent },
  { path: 'settings', component: UserSettingsComponent },
  { path: 'links', component: LinksListComponent },
  {
    path: 'admin',
    canActivate: [adminGuard],
    children: [
      { path: '', redirectTo: 'users', pathMatch: 'full' },
      { path: 'users', component: AdminUsersListComponent },
      { path: 'users/:id', component: AdminUserDetailComponent },
    ],
  },
];
