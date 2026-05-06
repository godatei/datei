import {
  ChangeDetectionStrategy,
  Component,
  effect,
  inject,
  input,
  output,
  signal,
} from '@angular/core';
import { form, FormField, FormRoot, required } from '@angular/forms/signals';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSnackBar, MatSnackBarModule } from '@angular/material/snack-bar';
import { firstValueFrom } from 'rxjs';
import type { UserDataPort } from './user-data.port';

@Component({
  selector: 'app-user-profile',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    FormField,
    FormRoot,
    MatButtonModule,
    MatCardModule,
    MatFormFieldModule,
    MatInputModule,
    MatSnackBarModule,
  ],
  templateUrl: './user-profile.component.html',
})
export class UserProfileComponent {
  private readonly snackBar = inject(MatSnackBar);

  readonly port = input.required<UserDataPort>();
  readonly name = input.required<string>();
  readonly changed = output<void>();

  readonly model = signal<{ name: string }>({ name: '' });

  constructor() {
    effect(() => {
      this.model.set({ name: this.name() });
    });
  }

  readonly profileForm = form(
    this.model,
    (p) => {
      required(p.name);
    },
    {
      submission: {
        action: async () => {
          const { name } = this.model();
          try {
            await firstValueFrom(this.port().updateName(name));
            this.snackBar.open('Profile updated', 'OK', { duration: 3000 });
            this.changed.emit();
          } catch {
            this.snackBar.open('Failed to update profile', 'OK', { duration: 3000 });
          }
        },
      },
    },
  );
}
