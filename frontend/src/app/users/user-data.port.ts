import { from, map, Observable, tap } from 'rxjs';
import { Api } from '~/api/api';
import {
  addUserEmailAdmin,
  disableUserMfaAdmin,
  getUserAdmin,
  listUserEmailsAdmin,
  removeUserEmailAdmin,
  resetUserPasswordAdmin,
  setPrimaryUserEmailAdmin,
  updateUserAdmin,
} from '~/api/functions';
import type { UserEmail } from '~/api/models/user-email';
import { AuthService } from '~/frontend/services/auth.service';
import { SettingsService } from '~/frontend/services/settings.service';

export interface UserSnapshot {
  name: string;
  isAdmin: boolean;
  mfaEnabled: boolean;
  archived: boolean;
}

export interface MfaSetupData {
  secret: string;
  qrCodeUrl: string;
}

export interface UserDataPort {
  load(): Observable<UserSnapshot>;
  listEmails(): Observable<UserEmail[]>;
  updateName(name: string): Observable<void>;
  changePassword(input: { currentPassword?: string; password: string }): Observable<void>;
  addEmail(email: string): Observable<void>;
  removeEmail(emailId: string): Observable<void>;
  setPrimaryEmail(emailId: string): Observable<void>;
  // MFA self flow — only used by `app-user-mfa`. Admin-side disable lives in `app-admin-mfa`.
  startMfaSetup(): Observable<MfaSetupData>;
  enableMfa(code: string): Observable<{ recoveryCodes: string[] }>;
  disableMfa(password: string): Observable<void>;
}

export function createSelfUserPort(settings: SettingsService, auth: AuthService): UserDataPort {
  return {
    load: () =>
      settings.getCurrentUser().pipe(
        map((u) => ({
          name: u.name,
          isAdmin: u.isAdmin,
          mfaEnabled: u.mfaEnabled,
          archived: false,
        })),
      ),
    listEmails: () => settings.getEmails(),
    updateName: (name) =>
      settings.updateUser({ name }).pipe(
        tap((res) => auth.updateName(res.name)),
        map(() => undefined),
      ),
    changePassword: ({ currentPassword, password }) =>
      settings.updateUser({ currentPassword, password }).pipe(map(() => undefined)),
    addEmail: (e) => settings.addEmail(e),
    removeEmail: (id) => settings.removeEmail(id),
    setPrimaryEmail: (id) => settings.setPrimaryEmail(id),
    startMfaSetup: () => settings.startMFASetup(),
    enableMfa: (code) => settings.enableMFA(code),
    disableMfa: (password) => settings.disableMFA(password),
  };
}

export function createAdminUserPort(api: Api, userId: string): UserDataPort {
  return {
    load: () =>
      from(api.invoke(getUserAdmin, { id: userId })).pipe(
        map((u) => ({
          name: u.name,
          isAdmin: u.isAdmin,
          mfaEnabled: u.mfaEnabled,
          archived: u.archived,
        })),
      ),
    listEmails: () =>
      from(api.invoke(listUserEmailsAdmin, { id: userId })).pipe(map((r) => r.emails)),
    updateName: (name) =>
      from(api.invoke(updateUserAdmin, { id: userId, body: { name } })).pipe(map(() => undefined)),
    // Admin reset ignores currentPassword.
    changePassword: ({ password }) =>
      from(api.invoke(resetUserPasswordAdmin, { id: userId, body: { password } })).pipe(
        map(() => undefined),
      ),
    addEmail: (email) =>
      from(api.invoke(addUserEmailAdmin, { id: userId, body: { email } })).pipe(
        map(() => undefined),
      ),
    removeEmail: (emailId) =>
      from(api.invoke(removeUserEmailAdmin, { id: userId, emailId })).pipe(map(() => undefined)),
    setPrimaryEmail: (emailId) =>
      from(api.invoke(setPrimaryUserEmailAdmin, { id: userId, emailId })).pipe(
        map(() => undefined),
      ),
    startMfaSetup: () => {
      throw new Error('startMfaSetup is not available in admin mode');
    },
    enableMfa: () => {
      throw new Error('enableMfa is not available in admin mode');
    },
    disableMfa: () =>
      from(api.invoke(disableUserMfaAdmin, { id: userId })).pipe(map(() => undefined)),
  };
}
