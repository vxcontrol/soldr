import { Pipe, PipeTransform } from '@angular/core';

export type SortFunc<T> = (language?: string) => (a: T, b: T) => number;

@Pipe({
    name: 'sort'
})
export class SortPipe implements PipeTransform {
    transform<T>(array: T[], func: SortFunc<T>, language?: string): T[] {
        return Array.isArray(array) ? [...array]?.sort(func(language)) : array;
    }
}
