import { inject, LOCALE_ID, Pipe, PipeTransform } from '@angular/core';
import { isThisYear, isToday } from 'date-fns';

@Pipe({ name: 'smartDate' })
export class SmartDatePipe implements PipeTransform {
  private readonly locale = inject(LOCALE_ID);

  transform(value: Date | string | null | undefined): string {
    if (!value) return '';
    const date = typeof value === 'string' ? new Date(value) : value;
    if (Number.isNaN(date.getTime())) return '';
    if (isToday(date)) {
      return new Intl.DateTimeFormat(this.locale, { timeStyle: 'short' }).format(date);
    }
    if (isThisYear(date)) {
      return new Intl.DateTimeFormat(this.locale, { day: 'numeric', month: 'short' }).format(date);
    }
    return new Intl.DateTimeFormat(this.locale, { dateStyle: 'medium' }).format(date);
  }
}
