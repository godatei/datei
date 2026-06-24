import {
  ApplicationConfig,
  inject,
  provideAppInitializer,
  provideBrowserGlobalErrorListeners,
} from '@angular/core';
import { provideHttpClient, withInterceptors, withXhr } from '@angular/common/http';
import { provideRouter } from '@angular/router';
import { MAT_ICON_DEFAULT_OPTIONS, MatIconRegistry } from '@angular/material/icon';
import { DomSanitizer } from '@angular/platform-browser';
import { registerFileLsicons } from '~/util/file-icons';

import { routes } from '~/frontend/app.routes';
import { tokenInterceptor } from '~/frontend/services/auth.service';
import { errorInterceptor } from '~/frontend/services/error.interceptor';
import { publicLinkTokenInterceptor } from '~/frontend/public-links/public-link-token.interceptor';

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    provideRouter(routes),
    // tokenInterceptor skips `/api/v1/public/*`, so publicLinkTokenInterceptor
    // can safely set the Authorization header on those routes without being
    // overwritten. errorInterceptor runs last so it sees the final response.
    provideHttpClient(
      withXhr(),
      withInterceptors([tokenInterceptor, publicLinkTokenInterceptor, errorInterceptor]),
    ),
    // Route every <mat-icon> through Material Symbols (variable) by default.
    { provide: MAT_ICON_DEFAULT_OPTIONS, useValue: { fontSet: 'material-symbols-outlined' } },
    // Register lsicon file-type SVGs so <mat-icon svgIcon="lsicon:..."> works.
    provideAppInitializer(() => registerFileLsicons(inject(MatIconRegistry), inject(DomSanitizer))),
  ],
};
