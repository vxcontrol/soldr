import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
    name: 'keys'
})
export class KeysPipe implements PipeTransform {
    transform(value: Record<string, any>): string[] {
        return Object.keys(value || {});
    }
}
