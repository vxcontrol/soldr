import { Pipe, PipeTransform } from '@angular/core';
import { DateTime } from 'luxon';

@Pipe({
    name: 'daysBefore'
})
export class DaysBeforePipe implements PipeTransform {
    transform(value: DateTime): number | undefined {
        return value.isValid ? Math.floor(value.diff(DateTime.now(), 'days').days) : undefined;
    }
}
