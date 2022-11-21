import { Pipe, PipeTransform } from '@angular/core';
import { DateFormatter } from '@ptsecurity/mosaic/core';
import { DateTime } from 'luxon';

@Pipe({
    name: 'absoluteShortDateTime'
})
export class AbsoluteShortDateTimePipe implements PipeTransform {
    constructor(private formatter: DateFormatter<DateTime>) {}

    transform(value: DateTime): string {
        return value && value.isValid ? this.formatter.absoluteShortDateTime(value) : undefined;
    }
}
