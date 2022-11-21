import { Pipe, PipeTransform } from '@angular/core';
import { DateTime } from 'luxon';

@Pipe({
    name: 'toDateTime'
})
export class ToDateTimePipe implements PipeTransform {
    transform(value: string, lang: string): DateTime {
        return DateTime.fromISO(value).setLocale(lang);
    }
}
