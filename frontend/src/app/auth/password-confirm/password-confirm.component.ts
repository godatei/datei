import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import {
  AbstractControl,
  FormControl,
  FormGroup,
  ReactiveFormsModule,
  ValidationErrors,
  Validators,
} from '@angular/forms';
import { ErrorStateMatcher } from '@angular/material/core';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';

export function passwordMatchValidator(control: AbstractControl): ValidationErrors | null {
  const password = control.get('password')?.value;
  const confirm = control.get('confirmPassword')?.value;
  return password === confirm ? null : { passwordMismatch: true };
}

export function passwordConfirmControls(): {
  password: [string, ((control: AbstractControl) => ValidationErrors | null)[]];
  confirmPassword: [string, (control: AbstractControl) => ValidationErrors | null];
} {
  return {
    password: ['', [Validators.required, Validators.minLength(8)]],
    confirmPassword: ['', Validators.required],
  };
}

/** Shows error state when the parent group has passwordMismatch and the control is touched. */
class PasswordMismatchStateMatcher implements ErrorStateMatcher {
  isErrorState(control: FormControl | null): boolean {
    if (!control) return false;
    const group = control.parent;
    return control.touched && !!group?.hasError('passwordMismatch');
  }
}

@Component({
  selector: 'app-password-confirm',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [ReactiveFormsModule, MatFormFieldModule, MatInputModule],
  templateUrl: './password-confirm.component.html',
  host: { style: 'display:contents' },
})
export class PasswordConfirmComponent {
  readonly group = input.required<FormGroup>();
  readonly passwordLabel = input('Password');
  readonly confirmLabel = input('Confirm password');
  readonly confirmErrorMatcher = new PasswordMismatchStateMatcher();
}
