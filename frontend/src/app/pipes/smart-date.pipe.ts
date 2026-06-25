import { inject, LOCALE_ID, Pipe, PipeTransform } from '@angular/core';
import { isThisYear, isToday } from 'date-fns';

@Pipe({ name: 'smartDate' })
export class SmartDatePipe implements PipeTransform {
  private readonly locale = inject(LOCALE_ID);

  // Constructing an Intl.DateTimeFormat is comparatively expensive, so build one
  // formatter per output style once and reuse it across every transform call.
  private readonly todayFormat = new Intl.DateTimeFormat(this.locale, { timeStyle: 'short' });
  private readonly thisYearFormat = new Intl.DateTimeFormat(this.locale, {
    day: 'numeric',
    month: 'short',
  });
  private readonly generalFormat = new Intl.DateTimeFormat(this.locale, { dateStyle: 'medium' });

  transform(value: Date | string | null | undefined): string {
    if (!value) return '';
    const date = typeof value === 'string' ? new Date(value) : value;
    if (Number.isNaN(date.getTime())) return '';
    if (isToday(date)) return this.todayFormat.format(date);
    if (isThisYear(date)) return this.thisYearFormat.format(date);
    return this.generalFormat.format(date);
  }
}
