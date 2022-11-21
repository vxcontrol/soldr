import { Pipe, PipeTransform } from '@angular/core';
// @ts-ignore
import * as Handlebars from 'handlebars/dist/cjs/handlebars';

@Pipe({
    name: 'hb'
})
export class HbPipe implements PipeTransform {
    transform(value: string, data: any): unknown {
        return Handlebars.compile(value || '')(data);
    }
}
