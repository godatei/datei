import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { MatSnackBar } from '@angular/material/snack-bar';
import { tap } from 'rxjs';
import { snackErrorDuration } from '~/frontend/constants';

export const errorInterceptor: HttpInterceptorFn = (req, next) => {
  const snackBar = inject(MatSnackBar);
  return next(req).pipe(
    tap({
      error: (e) => {
        if (e instanceof HttpErrorResponse && e.status === 0) {
          snackBar.open('Server unavailable. Please check your connection.', 'Dismiss', {
            duration: snackErrorDuration,
          });
        } else if (e instanceof HttpErrorResponse && e.status === 429) {
          snackBar.open('Too many requests. Please slow down.', 'Dismiss', {
            duration: snackErrorDuration,
          });
        }
      },
    }),
  );
};
