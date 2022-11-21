import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
    name: 'filter'
})
export class FilterPipe implements PipeTransform {
    transform(array: string[], filterValue: string): unknown {
        return array.filter((value) => value.toLowerCase().includes(filterValue || ''));
    }
}
