import { formatDate } from '@angular/common';
import { inject, LOCALE_ID, Pipe, PipeTransform } from '@angular/core';
import { isThisYear, isToday } from 'date-fns';

@Pipe({ name: 'smartDate' })
export class SmartDatePipe implements PipeTransform {
  private readonly locale = inject(LOCALE_ID);

  transform(value: Date | string | null | undefined): string {
    if (!value) return '';
    const date = typeof value === 'string' ? new Date(value) : value;
    if (Number.isNaN(date.getTime())) return '';
    if (isToday(date)) return formatDate(date, 'shortTime', this.locale);
    if (isThisYear(date))
      // Intl.DateTimeFormat keeps the order locale-correct: "Aug 23" (en-US), "23. Aug." (de).
      return new Intl.DateTimeFormat(this.locale, { day: 'numeric', month: 'short' }).format(date);
    return formatDate(date, 'mediumDate', this.locale);
  }
}
