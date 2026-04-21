import {
  HttpClient,
  HttpContextToken,
  HttpErrorResponse,
  HttpInterceptorFn,
  HttpRequest,
} from '@angular/common/http';
import { inject, Injectable, signal, computed } from '@angular/core';
import { jwtDecode } from 'jwt-decode';
import { Observable, tap, map, throwError } from 'rxjs';
import {
  login as loginFn,
  register as registerFn,
  resetPassword as resetPasswordFn,
  getLoginConfig as getLoginConfigFn,
} from '~/api/functions';
import type { LoginConfigResponse } from '~/api/models/login-config-response';

const tokenStorageKey = 'datei_token';
const actionTokenStorageKey = 'datei_action_token';

export const USE_ACTION_TOKEN = new HttpContextToken<boolean>(() => false);

export interface JWTClaims {
  sub: string;
  name: string;
  email: string;
  email_verified: boolean;
  action?: 'verify-email' | 'reset-password';
  exp: number;
  [claim: string]: unknown;
}

@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly httpClient = inject(HttpClient);
  private readonly _token = signal<string | null>(localStorage.getItem(tokenStorageKey));
  private readonly _nameOverride = signal<string | null>(null);

  readonly isAuthenticated = computed(() => {
    const claims = this.getClaims();
    return claims !== undefined && claims.exp * 1000 > Date.now();
  });

  readonly userName = computed(() => this._nameOverride() ?? this.getClaims()?.name);

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
    return loginFn(this.httpClient, '', { body: { email, password, mfaCode } }).pipe(
      map((r) => r.body),
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
    return registerFn(this.httpClient, '', { body: { email, name, password } }).pipe(
      map(() => undefined),
    );
  }

  resetPassword(email: string): Observable<void> {
    return resetPasswordFn(this.httpClient, '', { body: { email } }).pipe(map(() => undefined));
  }

  loginConfig(): Observable<LoginConfigResponse> {
    return getLoginConfigFn(this.httpClient, '').pipe(map((r) => r.body));
  }

  getClaims(): JWTClaims | undefined {
    const { claims } = this.getTokenAndClaims();
    return claims;
  }

  getSessionTokenAndClaims(): { token: string | null; claims: JWTClaims | undefined } {
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

  getActionTokenAndClaims(): { token: string | null; claims: JWTClaims | undefined } {
    const actionToken = this.actionToken;
    if (actionToken !== null) {
      try {
        return { token: actionToken, claims: jwtDecode<JWTClaims>(actionToken) };
      } catch {
        /* invalid token */
      }
    }
    return { token: null, claims: undefined };
  }

  getTokenAndClaims(): { token: string | null; claims: JWTClaims | undefined } {
    const action = this.getActionTokenAndClaims();
    if (action.claims) return action;
    return this.getSessionTokenAndClaims();
  }

  updateName(name: string): void {
    this._nameOverride.set(name);
  }

  logout(): void {
    this.token = null;
    this.actionToken = null;
    this._nameOverride.set(null);
  }
}

export const tokenInterceptor: HttpInterceptorFn = (req, next) => {
  const auth = inject(AuthService);
  if (authenticatedRoute(req)) {
    const useAction = req.context.get(USE_ACTION_TOKEN);
    const { token, claims } = useAction
      ? auth.getActionTokenAndClaims()
      : auth.getSessionTokenAndClaims();
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
  return !req.url.startsWith('/api/v1/auth');
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
