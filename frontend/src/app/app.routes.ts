import { Routes } from '@angular/router';
import {
  authGuard,
  emailVerificationGuard,
  jwtParamRedirectGuard,
  publicOnlyGuard,
} from '~/frontend/guards/auth.guard';
import { LoginComponent } from '~/frontend/auth/login/login.component';
import { RegisterComponent } from '~/frontend/auth/register/register.component';
import { ForgotComponent } from '~/frontend/auth/forgot/forgot.component';
import { VerifyComponent } from '~/frontend/auth/verify/verify.component';
import { ResetComponent } from '~/frontend/auth/reset/reset.component';

export const routes: Routes = [
  { path: 'login', canActivate: [publicOnlyGuard], component: LoginComponent },
  { path: 'register', canActivate: [publicOnlyGuard], component: RegisterComponent },
  { path: 'forgot', canActivate: [publicOnlyGuard], component: ForgotComponent },
  {
    path: '',
    canActivate: [jwtParamRedirectGuard, authGuard],
    children: [
      {
        path: 'verify',
        canActivate: [emailVerificationGuard],
        component: VerifyComponent,
      },
      { path: 'reset', component: ResetComponent },
      {
        path: '',
        loadComponent: () => import('~/frontend/nav/nav.component').then((m) => m.NavComponent),
        loadChildren: () => import('~/frontend/app-logged-in.routes').then((m) => m.routes),
      },
    ],
  },
  { path: '**', redirectTo: '/' },
];
