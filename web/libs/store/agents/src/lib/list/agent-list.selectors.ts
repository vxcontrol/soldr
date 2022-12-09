import { createFeatureSelector, createSelector } from '@ngrx/store';

import { filtrationToDictionary } from '@soldr/shared';

import * as fromAgents from './agent-list.reducer';
import { PrivateAgentCountResponse } from '@soldr/api';

export const selectAgentsState = createFeatureSelector<fromAgents.State>(fromAgents.agentListFeatureKey);
export const selectFilters = createSelector(selectAgentsState, (state) => state.filters);
export const selectFiltersCounters = createSelector(selectAgentsState, (state) => state.filtersCounters);
export const selectFiltersWithCounter = createSelector(selectFilters, selectFiltersCounters, (filters, counters) =>
    filters.map((filter) => ({
        id: filter.id,
        label: filter.label,
        value: filter.value,
        count: counters[filter.id as keyof PrivateAgentCountResponse] || 0
    }))
);
export const selectAgents = createSelector(selectAgentsState, (state) => state.agents);
export const selectTotal = createSelector(selectAgentsState, (state) => state.total);
export const selectIsLoadingAgents = createSelector(selectAgentsState, (state) => state.isLoadingAgents);
export const selectSelectedFilterId = createSelector(selectAgentsState, (state) => state.selectedFilterId);
export const selectSelectedGroupId = createSelector(selectAgentsState, (state) => state.selectedGroupId);
export const selectSelectedFilter = createSelector(selectFiltersWithCounter, selectSelectedFilterId, (filters, id) =>
    filters.find((item) => item.id === id)
);
export const selectRestored = createSelector(selectAgentsState, (state) => state.restored);
export const selectInitialized = createSelector(selectAgentsState, (state) => state.initialized);
export const selectGridFiltration = createSelector(selectAgentsState, (state) => state.gridFiltration);
export const selectGridFiltrationByField = createSelector(selectGridFiltration, (filtration) =>
    filtrationToDictionary(filtration)
);
export const selectGridSearch = createSelector(selectAgentsState, (state) => state.gridSearch);
export const selectGridSorting = createSelector(selectAgentsState, (state) => state.sorting);
export const selectVersions = createSelector(selectAgentsState, (state) => state.versions);
export const selectSelectedAgentsIds = createSelector(selectAgentsState, (state) => state.selectedIds);
export const selectSelectedAgents = createSelector(selectAgentsState, (state) =>
    state.agents.filter(({ id }) => state.selectedIds.includes(id))
);
export const selectSelectedAgent = createSelector(selectAgentsState, (state) => state.agent);
export const selectIsUpgradingAgents = createSelector(selectAgentsState, (state) => state.isUpgradingAgents);
export const selectIsUpgradingCancelAgent = createSelector(selectAgentsState, (state) => state.isCancelUpgradingAgent);
export const selectIsMovingAgents = createSelector(selectAgentsState, (state) => state.isMovingAgents);
export const selectIsUpdatingAgent = createSelector(selectAgentsState, (state) => state.isUpdatingAgents);
export const selectIsBlockingAgents = createSelector(selectAgentsState, (state) => state.isBlockingAgents);
export const selectIsDeletingAgents = createSelector(selectAgentsState, (state) => state.isDeletingAgents);
export const selectIsDeletingFromGroup = createSelector(selectAgentsState, (state) => state.isDeletingFromGroup);
export const selectPage = createSelector(selectAgentsState, (state) => state.page);
export const selectAgentModules = createSelector(selectAgentsState, (state) => state.selectedAgentModules);
export const selectInitializedAgent = createSelector(selectAgentsState, (state) => state.isInitializedAgent);
export const selectIsLoadingAgent = createSelector(selectAgentsState, (state) => state.isLoadingAgent);

export const selectIsUpdatingAgentData = createSelector(selectAgentsState, (state) => state.isUpdatingAgentData);

export const selectDeleteError = createSelector(selectAgentsState, (state) => state.deleteError);
export const selectMoveToGroupError = createSelector(selectAgentsState, (state) => state.moveToGroupError);
export const selectUpdateError = createSelector(selectAgentsState, (state) => state.updateError);

export const selectInitialListQuery = createSelector(
    selectSelectedFilter,
    selectSelectedGroupId,
    selectGridFiltration,
    selectGridSearch,
    selectGridSorting,
    (filter, groupId, gridFiltration, gridSearch, sorting) => {
        const filters = [
            ...(filter?.value || []),
            ...(groupId ? [{ field: 'group_id', value: [groupId] }] : []),
            ...gridFiltration,
            ...(gridSearch ? [{ field: 'data', value: gridSearch }] : [])
        ];

        return { filters, sort: sorting || {} };
    }
);

export const selectFilterItemsGroupIds = createSelector(selectAgentsState, (state) => state.filterItemsGroupIds);
export const selectFilterItemsModuleNames = createSelector(selectAgentsState, (state) => state.filterItemsModuleNames);
export const selectFilterItemsVersions = createSelector(selectAgentsState, (state) => state.filterItemsVersions);
export const selectFilterItemsOs = createSelector(selectAgentsState, (state) => state.filterItemsOs);
export const selectFilterItemsTags = createSelector(selectAgentsState, (state) => state.filterItemsTags);
