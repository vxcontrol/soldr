import { Filtration } from '@soldr/shared';

export function filtrationToDictionary(gridFiltration: Filtration[]) {
    return gridFiltration
        .filter((item) => (Array.isArray(item.value) ? !!item.value[0] : !!item.value))
        .reduce((acc, filter) => ({ ...acc, [filter.field]: filter }), {});
}
