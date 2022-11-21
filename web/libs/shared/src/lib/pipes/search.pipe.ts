import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
    name: 'search'
})
export class SearchPipe implements PipeTransform {
    transform(value: any[], search: string, callback: (item: any, search: string) => boolean): any[] {
        if (!value || !callback) {
            return value;
        }

        return value.filter((item) => callback(item, search));
    }
}
