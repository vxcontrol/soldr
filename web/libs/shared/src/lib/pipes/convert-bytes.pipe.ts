import { Pipe, PipeTransform } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';

const LEVEL = 1024;
const SIZES = ['Byte', 'KByte', 'MByte', 'GByte', 'TByte'];

@Pipe({
    name: 'convertBytes'
})
export class ConvertBytesPipe implements PipeTransform {
    constructor(private transloco: TranslocoService) {}

    transform(bytes: number): any {
        const exp = Math.floor(Math.log(bytes) / Math.log(LEVEL));
        const result = Math.round(bytes / Math.pow(LEVEL, exp));

        if (!bytes) {
            return `0 ${this.transloco.translate(`common.Common.Pseudo.Text.Byte`)}`;
        }

        if (exp > SIZES.length - 1) {
            return `${result} ${this.transloco.translate(`common.Common.Pseudo.Text.${SIZES[SIZES.length - 1]}`)}`;
        }

        return `${result} ${this.transloco.translate(`common.Common.Pseudo.Text.${SIZES[exp]}`)}`;
    }
}
