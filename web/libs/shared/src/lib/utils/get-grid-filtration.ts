import { Filtration } from '../types';

export const getGridFiltration = (filtration: Filtration, gridFiltration: Filtration[]) => {
    const needRemoveFiltration = Array.isArray(filtration.value) ? !filtration.value.length : !!filtration.value;
    const updatedFiltration = gridFiltration.filter((item: Filtration) => item.field !== filtration.field);

    return [...updatedFiltration, ...(needRemoveFiltration ? [] : [filtration])];
};
