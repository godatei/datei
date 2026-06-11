import { from, map, Observable, tap } from 'rxjs';
import { Api } from '~/api/api';
import {
  addUserEmailAdmin,
  getUserAdmin,
  listUserEmailsAdmin,
  removeUserEmailAdmin,
  resetUserPasswordAdmin,
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

export interface BaseUserPort {
  load(): Observable<UserSnapshot>;
  listEmails(): Observable<UserEmail[]>;
  updateName(name: string): Observable<void>;
  changePassword(input: { currentPassword?: string; password: string }): Observable<void>;
  addEmail(email: string): Observable<void>;
  removeEmail(emailId: string): Observable<void>;
  setPrimaryEmail(emailId: string): Observable<void>;
}

export interface SelfUserPort extends BaseUserPort {
  startMfaSetup(): Observable<MfaSetupData>;
  enableMfa(code: string): Observable<{ recoveryCodes: string[] }>;
  disableMfa(password: string): Observable<void>;
}

export function createSelfUserPort(settings: SettingsService, auth: AuthService): SelfUserPort {
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

export function createAdminUserPort(api: Api, userId: string): BaseUserPort {
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
      from(api.invoke(updateUserAdmin, { id: userId, body: { primaryEmailId: emailId } })).pipe(
        map(() => undefined),
      ),
  };
}
