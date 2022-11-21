import { Pipe, PipeTransform } from '@angular/core';
import { DateFormatter } from '@ptsecurity/mosaic/core';
import { DateTime } from 'luxon';

const MILLISECONDS_LENGTH = 4;
@Pipe({
    name: 'absoluteLongDateTimeWithSeconds'
})
export class AbsoluteLongDateTimeWithSecondsPipe implements PipeTransform {
    constructor(private formatter: DateFormatter<DateTime>) {}

    transform(value: DateTime): string {
        const dateWithMilliseconds = this.formatter.absoluteLongDateTime(value, { milliseconds: true });

        return value.isValid
            ? dateWithMilliseconds.slice(0, dateWithMilliseconds.length - MILLISECONDS_LENGTH)
            : undefined;
    }
}
