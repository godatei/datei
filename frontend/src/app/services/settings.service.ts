import { HttpClient, HttpContext } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable, map } from 'rxjs';
import {
  updateUser as updateUserFn,
  updateUserEmail as updateUserEmailFn,
  requestEmailVerification as requestEmailVerificationFn,
  confirmEmailVerification as confirmEmailVerificationFn,
  setupMfa as setupMfaFn,
  enableMfa as enableMfaFn,
  disableMfa as disableMfaFn,
  regenerateMfaRecoveryCodes as regenerateMfaRecoveryCodesFn,
  getMfaRecoveryCodesStatus as getMfaRecoveryCodesStatusFn,
  listEmails as listEmailsFn,
  addEmail as addEmailFn,
  removeEmail as removeEmailFn,
  setPrimaryEmail as setPrimaryEmailFn,
} from '~/api/functions';
import type { SetupMfaResponse } from '~/api/models/setup-mfa-response';
import type { EnableMfaResponse } from '~/api/models/enable-mfa-response';
import type { RegenerateMfaRecoveryCodesResponse } from '~/api/models/regenerate-mfa-recovery-codes-response';
import type { MfaRecoveryCodesStatusResponse } from '~/api/models/mfa-recovery-codes-status-response';
import type { UserEmail } from '~/api/models/user-email';
import { USE_ACTION_TOKEN } from './auth.service';

@Injectable({ providedIn: 'root' })
export class SettingsService {
  private readonly httpClient = inject(HttpClient);

  updateUser(
    request: { name?: string; password?: string },
    useActionToken = false,
  ): Observable<void> {
    const context = useActionToken ? new HttpContext().set(USE_ACTION_TOKEN, true) : undefined;
    return updateUserFn(this.httpClient, '', { body: request }, context).pipe(map(() => undefined));
  }

  requestEmailChange(email: string): Observable<void> {
    return updateUserEmailFn(this.httpClient, '', { body: { email } }).pipe(map(() => undefined));
  }

  requestEmailVerification(): Observable<void> {
    return requestEmailVerificationFn(this.httpClient, '').pipe(map(() => undefined));
  }

  confirmEmailVerification(): Observable<void> {
    const context = new HttpContext().set(USE_ACTION_TOKEN, true);
    return confirmEmailVerificationFn(this.httpClient, '', undefined, context).pipe(
      map(() => undefined),
    );
  }

  startMFASetup(): Observable<SetupMfaResponse> {
    return setupMfaFn(this.httpClient, '').pipe(map((r) => r.body));
  }

  enableMFA(code: string): Observable<EnableMfaResponse> {
    return enableMfaFn(this.httpClient, '', { body: { code } }).pipe(map((r) => r.body));
  }

  disableMFA(password: string): Observable<void> {
    return disableMfaFn(this.httpClient, '', { body: { password } }).pipe(map(() => undefined));
  }

  regenerateRecoveryCodes(password: string): Observable<RegenerateMfaRecoveryCodesResponse> {
    return regenerateMfaRecoveryCodesFn(this.httpClient, '', { body: { password } }).pipe(
      map((r) => r.body),
    );
  }

  getMFARecoveryCodesStatus(): Observable<MfaRecoveryCodesStatusResponse> {
    return getMfaRecoveryCodesStatusFn(this.httpClient, '').pipe(map((r) => r.body));
  }

  getEmails(): Observable<UserEmail[]> {
    return listEmailsFn(this.httpClient, '').pipe(map((r) => r.body.emails));
  }

  addEmail(email: string): Observable<void> {
    return addEmailFn(this.httpClient, '', { body: { email } }).pipe(map(() => undefined));
  }

  removeEmail(emailId: string): Observable<void> {
    return removeEmailFn(this.httpClient, '', { emailId }).pipe(map(() => undefined));
  }

  setPrimaryEmail(emailId: string): Observable<void> {
    return setPrimaryEmailFn(this.httpClient, '', { emailId }).pipe(map(() => undefined));
  }
}
