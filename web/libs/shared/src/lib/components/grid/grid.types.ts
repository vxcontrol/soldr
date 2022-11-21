import { ColDef, ColumnState } from 'ag-grid-community';

import { LocalizedValue } from '@soldr/shared';

export interface GridColumnFilterItem {
    label: string;
    value: string | number;
}

export interface GridColumnDef extends ColDef {
    autoSize?: boolean;
    default?: boolean;
    required?: boolean;
    sortField?: string;
    filtrationField?: string;
}

export interface GridColumnFilter {
    field?: string;
    items: GridColumnFilterItem[];
    multiple?: boolean;
    order?: number;
    placeholder?: string;
    title?: string;
    needTranslateLabel?: boolean;
}

export interface GridFilter {
    definition: GridColumnFilter;
    field: string;
    placeholder: LocalizedValue;
    selectedLabels: LocalizedValue[];
    title: LocalizedValue;
}

export interface Sorting {
    order: SortingDirection;
    prop: string;
}

export enum SortingDirection {
    ASC = 'ascending',
    DESC = 'descending'
}

export interface Pagination {
    page: number;
    pageSize: number;
}

export enum Selection {
    Multiple = 'multiple',
    Single = 'single'
}

export interface LocalizedData<T> {
    origin: T;
    localizedData: { [key: string]: string };
}

export type GridsState = { [storageKey: string]: ColumnState[] };
