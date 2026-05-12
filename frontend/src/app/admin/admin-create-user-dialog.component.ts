import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { email, form, FormField, FormRoot, required } from '@angular/forms/signals';
import { MatButtonModule } from '@angular/material/button';
import { MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { firstValueFrom } from 'rxjs';
import type { AdminUserListItem } from '~/api/models/admin-user-list-item';
import {
  PasswordConfirmComponent,
  passwordConfirmSchema,
} from '~/frontend/auth/password-confirm/password-confirm.component';
import { snackErrorDuration, snackSuccessDuration } from '~/frontend/constants';
import { AdminUsersService } from '~/frontend/services/admin-users.service';

@Component({
  selector: 'app-admin-create-user-dialog',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    FormField,
    FormRoot,
    MatButtonModule,
    MatDialogModule,
    MatFormFieldModule,
    MatInputModule,
    MatSlideToggleModule,
    MatSnackBarModule,
    PasswordConfirmComponent,
  ],
  templateUrl: './admin-create-user-dialog.component.html',
})
export class AdminCreateUserDialogComponent {
  private readonly admin = inject(AdminUsersService);
  private readonly dialogRef = inject(
    MatDialogRef<AdminCreateUserDialogComponent, AdminUserListItem>,
  );
  private readonly snackBar = inject(MatSnackBar);

  readonly model = signal<{
    name: string;
    email: string;
    password: string;
    confirmPassword: string;
    isAdmin: boolean;
  }>({ name: '', email: '', password: '', confirmPassword: '', isAdmin: false });

  readonly createForm = form(
    this.model,
    (p) => {
      required(p.name);
      required(p.email);
      email(p.email);
      passwordConfirmSchema(p.password, p.confirmPassword);
    },
    {
      submission: {
        action: async () => {
          const { name, email, password, isAdmin } = this.model();
          try {
            const created = await firstValueFrom(
              this.admin.createUser({ name, email, password, isAdmin }),
            );
            this.snackBar.open('User created', 'OK', { duration: snackSuccessDuration });
            this.dialogRef.close(created);
          } catch {
            this.snackBar.open('Failed to create user. Email may already be in use.', 'OK', {
              duration: snackErrorDuration,
            });
          }
        },
      },
    },
  );

  toggleAdmin(value: boolean) {
    this.model.update((m) => ({ ...m, isAdmin: value }));
  }

  cancel() {
    this.dialogRef.close();
  }
}
