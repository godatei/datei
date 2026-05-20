import { Pipe, PipeTransform } from '@angular/core';
import { formatBytes } from 'frontend/src/util/format-bytes';

@Pipe({ name: 'bytes' })
export class BytesPipe implements PipeTransform {
  transform(value: number | null | undefined): string {
    return formatBytes(value);
  }
}
