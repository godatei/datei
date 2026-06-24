import { Pipe, PipeTransform } from '@angular/core';
import { format, isThisYear, isToday } from 'date-fns';

@Pipe({ name: 'smartDate' })
export class SmartDatePipe implements PipeTransform {
  transform(value: Date | string | null | undefined): string {
    if (!value) return '';
    const date = typeof value === 'string' ? new Date(value) : value;
    if (Number.isNaN(date.getTime())) return '';
    if (isToday(date)) return format(date, 'HH:mm');
    if (isThisYear(date)) return format(date, 'd MMM');
    return format(date, 'd MMM yyyy');
  }
}
