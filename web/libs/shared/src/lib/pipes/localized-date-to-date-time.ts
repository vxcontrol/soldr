import { Pipe, PipeTransform } from '@angular/core';
import { DateAdapter } from '@ptsecurity/cdk/datetime';
import { DateFormatter } from '@ptsecurity/mosaic/core';
import { DateTime } from 'luxon';

@Pipe({
    name: 'localizedDateToDateTime'
})
export class LocalizedDateToDateTimePipe implements PipeTransform {
    constructor(private formatter: DateFormatter<DateTime>, private adapter: DateAdapter<DateTime>) {}

    transform(value: string, format: string, lang: string): DateTime {
        this.formatter.setLocale(lang);
        this.adapter.setLocale(lang);

        return DateTime.fromFormat(value, format).setLocale(lang);
    }
}
