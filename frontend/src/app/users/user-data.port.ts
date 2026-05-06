import { map, Observable, tap } from 'rxjs';
import type { UserEmail } from '~/api/models/user-email';
import { AdminUsersService } from '~/frontend/services/admin-users.service';
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

export function createAdminUserPort(admin: AdminUsersService, userId: string): UserDataPort {
  return {
    load: () =>
      admin.getUser(userId).pipe(
        map((u) => ({
          name: u.name,
          isAdmin: u.isAdmin,
          mfaEnabled: u.mfaEnabled,
          archived: u.archived,
        })),
      ),
    listEmails: () => admin.listEmails(userId),
    updateName: (name) => admin.updateUser(userId, { name }),
    // Admin reset ignores currentPassword.
    changePassword: ({ password }) => admin.resetPassword(userId, password),
    addEmail: (e) => admin.addEmail(userId, e),
    removeEmail: (id) => admin.removeEmail(userId, id),
    setPrimaryEmail: (id) => admin.setPrimaryEmail(userId, id),
    startMfaSetup: () => {
      throw new Error('startMfaSetup is not available in admin mode');
    },
    enableMfa: () => {
      throw new Error('enableMfa is not available in admin mode');
    },
    disableMfa: () => admin.disableMfa(userId),
  };
}
