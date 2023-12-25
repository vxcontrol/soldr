export type SimpleFiltrationValue = string | number | boolean;
export type ArrayFiltrationValue = SimpleFiltrationValue[];
export type FiltrationValue = SimpleFiltrationValue | ArrayFiltrationValue;

export interface Filtration {
    field: string;
    value: FiltrationValue;
}

export type GridFilters = { [key: string]: Filtration };
