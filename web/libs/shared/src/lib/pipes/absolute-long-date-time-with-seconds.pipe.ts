import { Pipe, PipeTransform } from '@angular/core';
import { DateFormatter } from '@ptsecurity/mosaic/core';
import { DateTime } from 'luxon';

@Pipe({
    name: 'absoluteLongDateTimeWithSeconds'
})
export class AbsoluteLongDateTimeWithSecondsPipe implements PipeTransform {
    constructor(private formatter: DateFormatter<DateTime>) {}

    transform(value: DateTime): string {
        return value.isValid ? this.formatter.absoluteLongDateTime(value, { seconds: true }) : undefined;
    }
}
