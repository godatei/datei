import {
  HttpClient,
  HttpErrorResponse,
  HttpInterceptorFn,
  HttpRequest,
} from '@angular/common/http';
import { inject, Injectable, signal, computed } from '@angular/core';
import { jwtDecode } from 'jwt-decode';
import { Observable, tap, map, throwError } from 'rxjs';

const tokenStorageKey = 'datei_token';
const actionTokenStorageKey = 'datei_action_token';
const authBaseUrl = '/api/v1/auth';

export interface JWTClaims {
  sub: string;
  name: string;
  email: string;
  email_verified: boolean;
  password_reset?: boolean;
  exp: number;
  [claim: string]: unknown;
}

interface LoginResponse {
  token?: string;
  requiresMfa?: boolean;
}

interface LoginConfig {
  registrationEnabled: boolean;
}

@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly httpClient = inject(HttpClient);
  private readonly _token = signal<string | null>(localStorage.getItem(tokenStorageKey));

  readonly isAuthenticated = computed(() => {
    const claims = this.getClaims();
    return claims !== undefined && claims.exp * 1000 > Date.now();
  });

  private get token(): string | null {
    return this._token();
  }

  private set token(value: string | null) {
    this._token.set(value);
    if (value !== null) {
      localStorage.setItem(tokenStorageKey, value);
    } else {
      localStorage.removeItem(tokenStorageKey);
    }
  }

  get actionToken(): string | null {
    return sessionStorage.getItem(actionTokenStorageKey);
  }

  set actionToken(value: string | null) {
    if (value !== null) {
      sessionStorage.setItem(actionTokenStorageKey, value);
    } else {
      sessionStorage.removeItem(actionTokenStorageKey);
    }
  }

  login(email: string, password: string, mfaCode?: string): Observable<{ requiresMfa: boolean }> {
    return this.httpClient
      .post<LoginResponse>(`${authBaseUrl}/login`, { email, password, mfaCode })
      .pipe(
        tap((r) => {
          if (!r.requiresMfa && r.token) {
            this.token = r.token;
            this.actionToken = null;
          }
        }),
        map((r) => ({ requiresMfa: r.requiresMfa ?? false })),
      );
  }

  register(email: string, name: string, password: string): Observable<void> {
    return this.httpClient.post<void>(`${authBaseUrl}/register`, { email, name, password });
  }

  resetPassword(email: string): Observable<void> {
    return this.httpClient.post<void>(`${authBaseUrl}/reset`, { email });
  }

  loginConfig(): Observable<LoginConfig> {
    return this.httpClient.get<LoginConfig>(`${authBaseUrl}/login/config`);
  }

  getClaims(): JWTClaims | undefined {
    const { claims } = this.getTokenAndClaims();
    return claims;
  }

  getTokenAndClaims(): { token: string | null; claims: JWTClaims | undefined } {
    const actionToken = this.actionToken;
    if (actionToken !== null) {
      try {
        return { token: actionToken, claims: jwtDecode<JWTClaims>(actionToken) };
      } catch {
        /* invalid token */
      }
    }
    const token = this.token;
    if (token !== null) {
      try {
        return { token, claims: jwtDecode<JWTClaims>(token) };
      } catch {
        /* invalid token */
      }
    }
    return { token: null, claims: undefined };
  }

  logout(): void {
    this.token = null;
    this.actionToken = null;
  }
}

export const tokenInterceptor: HttpInterceptorFn = (req, next) => {
  const auth = inject(AuthService);
  if (authenticatedRoute(req)) {
    const { token, claims } = auth.getTokenAndClaims();
    if (claims && claims.exp * 1000 > Date.now()) {
      return next(req.clone({ headers: req.headers.set('Authorization', `Bearer ${token}`) })).pipe(
        tap({
          error: (e) => {
            if (e instanceof HttpErrorResponse && e.status === 401) {
              auth.logout();
              redirectToLogin(claims.email);
            }
          },
        }),
      );
    } else {
      auth.logout();
      redirectToLogin(claims?.email);
      return throwError(() => new Error('no token or token has expired'));
    }
  }
  return next(req);
};

function authenticatedRoute(req: HttpRequest<unknown>): boolean {
  return !req.url.startsWith(authBaseUrl);
}

function redirectToLogin(email?: string) {
  const url = new URL(location.href);
  if (url.searchParams.has('jwt')) {
    url.searchParams.delete('jwt');
  }
  if (url.pathname === '/reset') {
    url.pathname = '/forgot';
    url.searchParams.append('reason', 'reset-expired');
  } else {
    url.pathname = '/login';
    url.searchParams.append('reason', 'session-expired');
  }
  if (email) {
    url.searchParams.append('email', email);
  }
  location.assign(url);
}
