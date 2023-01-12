import { createFeatureSelector, createSelector } from '@ngrx/store';

import { filtrationToDictionary } from '@soldr/shared';

import * as fromModulesInstances from './modules-instances.reducer';

export const selectModulesInstancesState = createFeatureSelector<fromModulesInstances.State>(
    fromModulesInstances.modulesInstancesFeatureKey
);

export const selectViewMode = createSelector(selectModulesInstancesState, (state) => state.viewMode);
export const selectEntityId = createSelector(selectModulesInstancesState, (state) => state.entityId);
export const selectModuleName = createSelector(selectModulesInstancesState, (state) => state.moduleName);

export const selectEvents = createSelector(selectModulesInstancesState, (state) => state.events);
export const selectTotalEvents = createSelector(selectModulesInstancesState, (state) => state.totalEvents);
export const selectEventsPage = createSelector(selectModulesInstancesState, (state) => state.eventsPage);
export const selectEventsGridSearch = createSelector(selectModulesInstancesState, (state) => state.eventsGridSearch);
export const selectIsLoadingEvents = createSelector(selectModulesInstancesState, (state) => state.isLoadingEvents);
export const selectEventsGridFiltration = createSelector(
    selectModulesInstancesState,
    (state) => state.eventsGridFiltration
);
export const selectEventsGridSorting = createSelector(selectModulesInstancesState, (state) => state.eventsSorting);
export const selectEventsGridFiltrationByField = createSelector(selectEventsGridFiltration, (filtration) =>
    filtrationToDictionary(filtration)
);

export const selectModuleVersions = createSelector(selectModulesInstancesState, (state) => state.moduleVersions);
export const selectIsLoadingModuleVersions = createSelector(
    selectModulesInstancesState,
    (state) => state.isLoadingModuleVersions
);

export const selectIsEnablingModule = createSelector(selectModulesInstancesState, (state) => state.isEnablingModule);
export const selectIsDisablingModule = createSelector(selectModulesInstancesState, (state) => state.isDisablingModule);
export const selectIsDeletingModule = createSelector(selectModulesInstancesState, (state) => state.isDeletingModule);
export const selectIsSavingModule = createSelector(selectModulesInstancesState, (state) => state.isSavingModuleConfig);
export const selectIsUpdatingModule = createSelector(selectModulesInstancesState, (state) => state.isUpdatingModule);
export const selectIsChangingVersionModule = createSelector(
    selectModulesInstancesState,
    (state) => state.isChangingVersion
);

export const selectChangeVersionError = createSelector(
    selectModulesInstancesState,
    (state) => state.changeVersionError
);
export const selectDeleteModuleError = createSelector(selectModulesInstancesState, (state) => state.deleteModuleError);
export const selectDisableModuleError = createSelector(
    selectModulesInstancesState,
    (state) => state.disableModuleError
);
export const selectEnableModuleError = createSelector(selectModulesInstancesState, (state) => state.enableModuleError);
export const selectUpdateModuleError = createSelector(selectModulesInstancesState, (state) => state.updateModuleError);
export const selectSavingModuleError = createSelector(selectModulesInstancesState, (state) => state.savingModuleError);

export const selectEventFilterItemAgentNames = createSelector(
    selectModulesInstancesState,
    (state) => state.eventFilterItemAgentNames
);
export const selectEventFilterItemGroupIds = createSelector(
    selectModulesInstancesState,
    (state) => state.eventFilterItemGroupIds
);
