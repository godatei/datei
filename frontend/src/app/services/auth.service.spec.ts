import { TestBed } from '@angular/core/testing';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { HttpClient, provideHttpClient, withInterceptors } from '@angular/common/http';
import { AuthService, tokenInterceptor } from './auth.service';

function makeJwt(claims: Record<string, unknown>): string {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }));
  const payload = btoa(JSON.stringify(claims));
  return `${header}.${payload}.fake-signature`;
}

describe('tokenInterceptor', () => {
  let httpClient: HttpClient;
  let httpTesting: HttpTestingController;
  let auth: AuthService;

  beforeEach(() => {
    localStorage.clear();
    sessionStorage.clear();

    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(withInterceptors([tokenInterceptor])),
        provideHttpClientTesting(),
      ],
    });

    httpClient = TestBed.inject(HttpClient);
    httpTesting = TestBed.inject(HttpTestingController);
    auth = TestBed.inject(AuthService);
  });

  afterEach(() => {
    httpTesting.verify();
    localStorage.clear();
    sessionStorage.clear();
  });

  it('should attach Bearer token to non-auth API calls', () => {
    const token = makeJwt({ sub: '1', exp: Math.floor(Date.now() / 1000) + 3600 });
    localStorage.setItem('datei_token', token);
    // Force signal update
    auth['_token'].set(token);

    httpClient.get('/api/v1/settings/user').subscribe();

    const req = httpTesting.expectOne('/api/v1/settings/user');
    expect(req.request.headers.get('Authorization')).toBe(`Bearer ${token}`);
    req.flush({});
  });

  it('should skip auth header for /api/v1/auth/* routes', () => {
    const token = makeJwt({ sub: '1', exp: Math.floor(Date.now() / 1000) + 3600 });
    localStorage.setItem('datei_token', token);
    auth['_token'].set(token);

    httpClient.get('/api/v1/auth/login/config').subscribe();

    const req = httpTesting.expectOne('/api/v1/auth/login/config');
    expect(req.request.headers.has('Authorization')).toBe(false);
    req.flush({});
  });

  it('should not send action tokens for non-settings routes', () => {
    const sessionToken = makeJwt({
      sub: '1',
      name: 'Test',
      exp: Math.floor(Date.now() / 1000) + 3600,
    });
    const actionToken = makeJwt({
      sub: '1',
      password_reset: true,
      exp: Math.floor(Date.now() / 1000) + 3600,
    });
    localStorage.setItem('datei_token', sessionToken);
    sessionStorage.setItem('datei_action_token', actionToken);
    auth['_token'].set(sessionToken);

    httpClient.get('/api/v1/datei').subscribe();

    const req = httpTesting.expectOne('/api/v1/datei');
    expect(req.request.headers.get('Authorization')).toBe(`Bearer ${sessionToken}`);
    req.flush({});
  });
});
