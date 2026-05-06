import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable, map } from 'rxjs';
import {
  listUsersAdmin,
  createUserAdmin,
  getUserAdmin,
  updateUserAdmin,
  resetUserPasswordAdmin,
  listUserEmailsAdmin,
  addUserEmailAdmin,
  removeUserEmailAdmin,
  setPrimaryUserEmailAdmin,
  disableUserMfaAdmin,
  archiveUserAdmin,
  unarchiveUserAdmin,
} from '~/api/functions';
import type { AdminUserListItem } from '~/api/models/admin-user-list-item';
import type { UserEmail } from '~/api/models/user-email';

@Injectable({ providedIn: 'root' })
export class AdminUsersService {
  private readonly httpClient = inject(HttpClient);

  listUsers(): Observable<AdminUserListItem[]> {
    return listUsersAdmin(this.httpClient, '').pipe(map((r) => r.body.users));
  }

  createUser(body: {
    name: string;
    email: string;
    password: string;
    isAdmin: boolean;
  }): Observable<AdminUserListItem> {
    return createUserAdmin(this.httpClient, '', { body }).pipe(map((r) => r.body));
  }

  getUser(id: string): Observable<AdminUserListItem> {
    return getUserAdmin(this.httpClient, '', { id }).pipe(map((r) => r.body));
  }

  updateUser(id: string, body: { name?: string; isAdmin?: boolean }): Observable<void> {
    return updateUserAdmin(this.httpClient, '', { id, body }).pipe(map(() => undefined));
  }

  resetPassword(id: string, password: string): Observable<void> {
    return resetUserPasswordAdmin(this.httpClient, '', { id, body: { password } }).pipe(
      map(() => undefined),
    );
  }

  listEmails(id: string): Observable<UserEmail[]> {
    return listUserEmailsAdmin(this.httpClient, '', { id }).pipe(map((r) => r.body.emails));
  }

  addEmail(id: string, email: string): Observable<void> {
    return addUserEmailAdmin(this.httpClient, '', { id, body: { email } }).pipe(
      map(() => undefined),
    );
  }

  removeEmail(id: string, emailId: string): Observable<void> {
    return removeUserEmailAdmin(this.httpClient, '', { id, emailId }).pipe(map(() => undefined));
  }

  setPrimaryEmail(id: string, emailId: string): Observable<void> {
    return setPrimaryUserEmailAdmin(this.httpClient, '', { id, emailId }).pipe(
      map(() => undefined),
    );
  }

  disableMfa(id: string): Observable<void> {
    return disableUserMfaAdmin(this.httpClient, '', { id }).pipe(map(() => undefined));
  }

  archive(id: string): Observable<void> {
    return archiveUserAdmin(this.httpClient, '', { id }).pipe(map(() => undefined));
  }

  unarchive(id: string): Observable<void> {
    return unarchiveUserAdmin(this.httpClient, '', { id }).pipe(map(() => undefined));
  }
}
