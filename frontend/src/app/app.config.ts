import { ApplicationConfig, provideBrowserGlobalErrorListeners } from '@angular/core';
import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { provideRouter } from '@angular/router';

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
      withInterceptors([tokenInterceptor, publicLinkTokenInterceptor, errorInterceptor]),
    ),
  ],
};
