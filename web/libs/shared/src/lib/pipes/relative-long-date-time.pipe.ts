import { Pipe, PipeTransform } from '@angular/core';
import { DateFormatter } from '@ptsecurity/mosaic/core';
import { DateTime } from 'luxon';

@Pipe({
    name: 'relativeLongDateTime'
})
export class RelativeLongDateTimePipe implements PipeTransform {
    constructor(private formatter: DateFormatter<DateTime>) {}

    transform(value: DateTime): string {
        return value.isValid ? this.formatter.relativeLongDateTime(value) : undefined;
    }
}
