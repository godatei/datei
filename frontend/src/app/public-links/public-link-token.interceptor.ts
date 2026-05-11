import { HttpContextToken, HttpInterceptorFn } from '@angular/common/http';

// Per-request token carrying the public-link session JWT. Callers attach this
// to the HttpContext when invoking list/download under an unlocked public
// link; the interceptor below copies it onto the Authorization header.
export const PUBLIC_LINK_TOKEN = new HttpContextToken<string | undefined>(() => undefined);

export const publicLinkTokenInterceptor: HttpInterceptorFn = (req, next) => {
  const token = req.context.get(PUBLIC_LINK_TOKEN);
  if (!token) return next(req);
  return next(req.clone({ headers: req.headers.set('Authorization', `Bearer ${token}`) }));
};
