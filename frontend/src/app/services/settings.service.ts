import { HttpClient, HttpContext } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable, map } from 'rxjs';
import {
  confirmResetPassword,
  getCurrentUser,
  updateUser,
  updateUserEmail,
  requestEmailVerification,
  confirmEmailVerification,
  setupMfa,
  enableMfa,
  disableMfa,
  regenerateMfaRecoveryCodes,
  getMfaRecoveryCodesStatus,
  listEmails,
  addEmail,
  removeEmail,
  setPrimaryEmail,
} from '~/api/functions';
import type { SetupMfaResponse } from '~/api/models/setup-mfa-response';
import type { EnableMfaResponse } from '~/api/models/enable-mfa-response';
import type { RegenerateMfaRecoveryCodesResponse } from '~/api/models/regenerate-mfa-recovery-codes-response';
import type { MfaRecoveryCodesStatusResponse } from '~/api/models/mfa-recovery-codes-status-response';
import type { UserResponse } from '~/api/models/user-response';
import type { UserEmail } from '~/api/models/user-email';
import { USE_ACTION_TOKEN } from './auth.service';

@Injectable({ providedIn: 'root' })
export class SettingsService {
  private readonly httpClient = inject(HttpClient);

  confirmResetPassword(password: string): Observable<void> {
    const context = new HttpContext().set(USE_ACTION_TOKEN, true);
    return confirmResetPassword(this.httpClient, '', { body: { password } }, context).pipe(
      map(() => undefined),
    );
  }

  getCurrentUser(): Observable<UserResponse> {
    return getCurrentUser(this.httpClient, '').pipe(map((r) => r.body));
  }

  updateUser(request: {
    name?: string;
    password?: string;
    currentPassword?: string;
  }): Observable<UserResponse> {
    return updateUser(this.httpClient, '', { body: request }).pipe(map((r) => r.body));
  }

  updatePrimaryEmail(email: string): Observable<void> {
    return updateUserEmail(this.httpClient, '', { body: { email } }).pipe(map(() => undefined));
  }

  requestEmailVerification(): Observable<void> {
    return requestEmailVerification(this.httpClient, '').pipe(map(() => undefined));
  }

  confirmEmailVerification(): Observable<void> {
    const context = new HttpContext().set(USE_ACTION_TOKEN, true);
    return confirmEmailVerification(this.httpClient, '', undefined, context).pipe(
      map(() => undefined),
    );
  }

  startMFASetup(): Observable<SetupMfaResponse> {
    return setupMfa(this.httpClient, '').pipe(map((r) => r.body));
  }

  enableMFA(code: string): Observable<EnableMfaResponse> {
    return enableMfa(this.httpClient, '', { body: { code } }).pipe(map((r) => r.body));
  }

  disableMFA(password: string): Observable<void> {
    return disableMfa(this.httpClient, '', { body: { password } }).pipe(map(() => undefined));
  }

  regenerateRecoveryCodes(password: string): Observable<RegenerateMfaRecoveryCodesResponse> {
    return regenerateMfaRecoveryCodes(this.httpClient, '', { body: { password } }).pipe(
      map((r) => r.body),
    );
  }

  getMFARecoveryCodesStatus(): Observable<MfaRecoveryCodesStatusResponse> {
    return getMfaRecoveryCodesStatus(this.httpClient, '').pipe(map((r) => r.body));
  }

  getEmails(): Observable<UserEmail[]> {
    return listEmails(this.httpClient, '').pipe(map((r) => r.body.emails));
  }

  addEmail(email: string): Observable<void> {
    return addEmail(this.httpClient, '', { body: { email } }).pipe(map(() => undefined));
  }

  removeEmail(emailId: string): Observable<void> {
    return removeEmail(this.httpClient, '', { emailId }).pipe(map(() => undefined));
  }

  setPrimaryEmail(emailId: string): Observable<void> {
    return setPrimaryEmail(this.httpClient, '', { emailId }).pipe(map(() => undefined));
  }
}
