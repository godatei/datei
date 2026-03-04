import { inject } from '@angular/core';
import {
  ActivatedRouteSnapshot,
  CanActivateFn,
  createUrlTreeFromSnapshot,
  Router,
  RouterStateSnapshot,
} from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { AuthService } from '~/frontend/services/auth.service';
import { SettingsService } from '~/frontend/services/settings.service';

export const jwtParamRedirectGuard: CanActivateFn = (route: ActivatedRouteSnapshot) => {
  const auth = inject(AuthService);
  const jwt = route.queryParamMap.get('jwt');
  if (jwt === null) {
    return true;
  }
  auth.actionToken = jwt;
  const newTree = createUrlTreeFromSnapshot(route, [], null, null);
  delete newTree.queryParams['jwt'];
  return newTree;
};

export const authGuard: CanActivateFn = (_: ActivatedRouteSnapshot, state: RouterStateSnapshot) => {
  const auth = inject(AuthService);
  const router = inject(Router);
  const claims = auth.getClaims();

  if (!claims) {
    return router.createUrlTree(['/login']);
  }
  if (claims.password_reset) {
    return state.url === '/reset' ? true : router.createUrlTree(['/reset']);
  }
  if (!claims.email_verified) {
    return state.url === '/verify' ? true : router.createUrlTree(['/verify']);
  }
  return true;
};

export const publicOnlyGuard: CanActivateFn = () => {
  const auth = inject(AuthService);
  const router = inject(Router);
  return auth.isAuthenticated() ? router.createUrlTree(['/']) : true;
};

export const emailVerificationGuard: CanActivateFn = async () => {
  const auth = inject(AuthService);
  const settings = inject(SettingsService);
  const router = inject(Router);
  const claims = auth.getClaims();

  if (claims?.email_verified) {
    await firstValueFrom(settings.confirmEmailVerification());
    auth.logout();
    return router.createUrlTree(['/login'], { queryParams: { email: claims.email } });
  }
  return true;
};
