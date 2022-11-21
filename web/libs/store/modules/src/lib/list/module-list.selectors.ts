import { createFeatureSelector, createSelector } from '@ngrx/store';

import { filtrationToDictionary } from '@soldr/shared';

import * as fromModules from './module-list.reducer';

export const selectModulesState = createFeatureSelector<fromModules.State>(fromModules.moduleListFeatureKey);
export const selectModules = createSelector(selectModulesState, (state) => state.modules);
export const selectTotal = createSelector(selectModulesState, (state) => state.total);
export const selectIsLoadingModules = createSelector(selectModulesState, (state) => state.isLoadingModules);

export const selectGridFiltration = createSelector(selectModulesState, (state) => state.gridFiltration);
export const selectGridSearch = createSelector(selectModulesState, (state) => state.gridSearch);
export const selectGridSorting = createSelector(selectModulesState, (state) => state.sorting);

export const selectSelectedModules = createSelector(selectModulesState, (state) =>
    state.modules.filter(({ id }) => state.selectedIds.includes(id))
);
export const selectGridFiltrationByField = createSelector(selectGridFiltration, (filtration) =>
    filtrationToDictionary(filtration)
);
export const selectPage = createSelector(selectModulesState, (state) => state.page);
export const selectModulesTags = createSelector(selectModulesState, (state) => state.modulesTags);

export const selectCreateError = createSelector(selectModulesState, (state) => state.createError);
export const selectIsDeletingModule = createSelector(selectModulesState, (state) => state.isDeleteNodule);
export const selectDeleteError = createSelector(selectModulesState, (state) => state.deleteError);
export const selectIsDeletingModuleVersion = createSelector(selectModulesState, (state) => state.isDeleteNoduleVersion);
export const selectDeleteVersionError = createSelector(selectModulesState, (state) => state.deleteVersionError);
export const selectImportError = createSelector(selectModulesState, (state) => state.importError);
export const selectCreatedModule = createSelector(selectModulesState, (state) => state.createdModule);
export const selectIsCreatingModule = createSelector(selectModulesState, (state) => state.isCreatingModule);
export const selectIsImportingModule = createSelector(selectModulesState, (state) => state.isImportingModule);

export const selectVersions = createSelector(selectModulesState, (state) => state.versions);
export const selectIsLoadingModuleVersions = createSelector(selectModulesState, (state) => state.isLoadingVersions);

export const selectRestored = createSelector(selectModulesState, (state) => state.restored);

export const selectInitialListQuery = createSelector(
    selectGridFiltration,
    selectGridSearch,
    selectGridSorting,
    (gridFiltration, gridSearch, sorting) => {
        const filters = [...gridFiltration, ...(gridSearch ? [{ field: 'data', value: gridSearch }] : [])];

        return { filters, sort: sorting || {} };
    }
);
