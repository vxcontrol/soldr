import { Filtration } from './filtration';
import { LocaleItem } from './locale-item';

export interface Filter {
    id: string;
    label: string;
    count: number;
    countFields?: string[];
    value: Filtration[];
}

export interface FilterByGroup {
    id: string;
    hash?: string;
    label: LocaleItem;
    count: number;
}
