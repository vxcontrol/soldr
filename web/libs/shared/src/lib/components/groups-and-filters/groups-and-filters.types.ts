import { Observable } from 'rxjs';

import { Filtration } from '@soldr/shared';

export type GroupsAndFiltersListItem = FiltersItem | GroupsItem;

export interface FiltersItem {
    id: string;
    label: string | Observable<string>;
    value: Filtration[];
}

export interface GroupsItem {
    id: number;
    label: string | Observable<string>;
    value: string;
}
