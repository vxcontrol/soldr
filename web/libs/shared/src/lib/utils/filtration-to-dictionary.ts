import { Filtration, GridFilters } from '@soldr/shared';

export function filtrationToDictionary(gridFiltration: Filtration[]): GridFilters {
    return gridFiltration
        .filter((item) => (Array.isArray(item.value) ? !!item.value[0] : !!item.value))
        .reduce((acc, filter) => ({ ...acc, [filter.field]: filter }), {});
}
