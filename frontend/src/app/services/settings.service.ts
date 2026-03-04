import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';

const baseUrl = '/api/v1/settings';

interface SetupMFAResponse {
  secret: string;
  qrCodeUrl: string;
}

interface EnableMFAResponse {
  recoveryCodes: string[];
}

interface RegenerateMFARecoveryCodesResponse {
  recoveryCodes: string[];
}

interface MFARecoveryCodesStatusResponse {
  remainingCodes: number;
}

@Injectable({ providedIn: 'root' })
export class SettingsService {
  private readonly httpClient = inject(HttpClient);

  updateUser(request: { name?: string; password?: string }): Observable<void> {
    return this.httpClient.post<void>(`${baseUrl}/user`, request);
  }

  requestEmailChange(email: string): Observable<void> {
    return this.httpClient.patch<void>(`${baseUrl}/user/email`, { email });
  }

  requestEmailVerification(): Observable<void> {
    return this.httpClient.post<void>(`${baseUrl}/verify/request`, {});
  }

  confirmEmailVerification(): Observable<void> {
    return this.httpClient.post<void>(`${baseUrl}/verify/confirm`, {});
  }

  startMFASetup(): Observable<SetupMFAResponse> {
    return this.httpClient.post<SetupMFAResponse>(`${baseUrl}/mfa/setup`, {});
  }

  enableMFA(code: string): Observable<EnableMFAResponse> {
    return this.httpClient.post<EnableMFAResponse>(`${baseUrl}/mfa/enable`, { code });
  }

  disableMFA(password: string): Observable<void> {
    return this.httpClient.post<void>(`${baseUrl}/mfa/disable`, { password });
  }

  regenerateRecoveryCodes(password: string): Observable<RegenerateMFARecoveryCodesResponse> {
    return this.httpClient.post<RegenerateMFARecoveryCodesResponse>(
      `${baseUrl}/mfa/recovery-codes/regenerate`,
      { password },
    );
  }

  getMFARecoveryCodesStatus(): Observable<MFARecoveryCodesStatusResponse> {
    return this.httpClient.get<MFARecoveryCodesStatusResponse>(
      `${baseUrl}/mfa/recovery-codes/status`,
    );
  }
}
