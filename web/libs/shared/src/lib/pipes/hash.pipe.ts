import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
    name: 'hash'
})
export class HashPipe implements PipeTransform {
    transform(value: string): unknown {
        // eslint-disable-next-line @typescript-eslint/no-magic-numbers
        return value ? `${value.slice(0, 4)} ${value.slice(4, 8)} … ${value.slice(-8, -4)} ${value.slice(-4)}` : '';
    }
}
