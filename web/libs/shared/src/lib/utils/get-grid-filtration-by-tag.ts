import { Filtration } from '../types';

export const getGridFiltrationByTag = (gridFiltration: Filtration[], tag: string): Filtration[] =>
    gridFiltration.filter((item: Filtration) => item.field !== 'tags').concat({ field: 'tags', value: [tag] });
