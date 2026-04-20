import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import {
  type FieldTree,
  type SchemaPath,
  type SchemaPathRules,
  type PathKind,
  FormField,
  minLength,
  required,
  validate,
} from '@angular/forms/signals';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';

export function passwordConfirmSchema(
  passwordPath: SchemaPath<string, SchemaPathRules.Supported, PathKind>,
  confirmPath: SchemaPath<string, SchemaPathRules.Supported, PathKind>,
) {
  required(passwordPath);
  minLength(passwordPath, 8);
  required(confirmPath);
  validate(confirmPath, ({ value, valueOf }) => {
    const pw = valueOf(passwordPath);
    return value() !== pw ? { kind: 'mismatch', message: 'Passwords do not match' } : undefined;
  });
}

@Component({
  selector: 'app-password-confirm',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [FormField, MatFormFieldModule, MatInputModule],
  templateUrl: './password-confirm.component.html',
  host: { style: 'display:contents' },
})
export class PasswordConfirmComponent {
  readonly password = input.required<FieldTree<string>>();
  readonly confirmPassword = input.required<FieldTree<string>>();
  readonly passwordLabel = input('Password');
  readonly confirmLabel = input('Confirm password');
}
