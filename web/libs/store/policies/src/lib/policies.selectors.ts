import { createFeatureSelector, createSelector } from '@ngrx/store';

import { filtrationToDictionary } from '@soldr/shared';

import * as fromPolicies from './policies.reducer';

export const selectPoliciesState = createFeatureSelector<fromPolicies.State>(fromPolicies.policiesFeatureKey);
export const selectPolicies = createSelector(selectPoliciesState, (state) => state.policies);
export const selectFilters = createSelector(selectPoliciesState, (state) => state.filters);
export const selectFiltersCounters = createSelector(selectPoliciesState, (state) => state.filtersCounters);
export const selectFiltersWithCounter = createSelector(selectFilters, selectFiltersCounters, (filters, counters) =>
    filters.map((filter) => ({
        id: filter.id,
        label: filter.label,
        value: filter.value,
        count: counters[filter.id] || 0
    }))
);
export const selectSelectedFilterId = createSelector(selectPoliciesState, (state) => state.selectedFilterId);
export const selectSelectedGroupId = createSelector(selectPoliciesState, (state) => state.selectedGroupId);
export const selectSelectedFilter = createSelector(selectFiltersWithCounter, selectSelectedFilterId, (filters, id) =>
    filters.find((item) => item.id === id)
);
export const selectPolicy = createSelector(selectPoliciesState, (state) => state.policy);
export const selectCreatedPolicy = createSelector(selectPoliciesState, (state) => state.createdPolicy);
export const selectTotal = createSelector(selectPoliciesState, (state) => state.total);
export const selectSelectedPolicyId = createSelector(selectPoliciesState, (state) => state.selectedPolicyId);
export const selectSelectedPolicy = createSelector(selectPolicies, selectSelectedPolicyId, (policies, policyId) =>
    policies.find((policy) => policy.id.toString() === policyId)
);
export const selectSelectedPoliciesIds = createSelector(selectPoliciesState, (state) => state.selectedIds);
export const selectSelectedPolicies = createSelector(selectPoliciesState, (state) =>
    state.policies.filter(({ id }) => state.selectedIds.includes(id))
);
export const selectRestored = createSelector(selectPoliciesState, (state) => state.restored);
export const selectInitialized = createSelector(selectPoliciesState, (state) => state.initialized);
export const selectIsCopyingPolicy = createSelector(selectPoliciesState, (state) => state.isCopyingPolicy);
export const selectIsCreatingPolicy = createSelector(selectPoliciesState, (state) => state.isCreatingPolicy);
export const selectIsDeletingPolicy = createSelector(selectPoliciesState, (state) => state.isDeletingPolicy);
export const selectIsLinkingPolicy = createSelector(selectPoliciesState, (state) => state.isLinkingPolicy);
export const selectIsLoadingPolicy = createSelector(selectPoliciesState, (state) => state.isLoadingPolicy);
export const selectIsLoadingPolicies = createSelector(selectPoliciesState, (state) => state.isLoadingPolicies);
export const selectIsUnlinkingPolicy = createSelector(selectPoliciesState, (state) => state.isUnlinkingPolicy);
export const selectIsUpdatingPolicy = createSelector(selectPoliciesState, (state) => state.isUpdatingPolicy);
export const selectGridFiltration = createSelector(selectPoliciesState, (state) => state.gridFiltration);
export const selectGridFiltrationByField = createSelector(selectGridFiltration, (gridFiltration) =>
    gridFiltration
        .filter((item) => (Array.isArray(item.value) ? !!item.value[0] : !!item.value))
        .reduce((acc, filter) => ({ ...acc, [filter.field]: filter }), {})
);
export const selectGridSearch = createSelector(selectPoliciesState, (state) => state.gridSearch);
export const selectGridSorting = createSelector(selectPoliciesState, (state) => state.sorting);
export const selectPage = createSelector(selectPoliciesState, (state) => state.page);
export const selectPolicyModules = createSelector(selectPoliciesState, (state) => state.modulesOfPolicy);

export const selectCopyError = createSelector(selectPoliciesState, (state) => state.copyError);
export const selectCreateError = createSelector(selectPoliciesState, (state) => state.createError);
export const selectDeleteError = createSelector(selectPoliciesState, (state) => state.deleteError);
export const selectUpdateError = createSelector(selectPoliciesState, (state) => state.updateError);
export const selectLinkPolicyFromGroupError = createSelector(
    selectPoliciesState,
    (state) => state.linkPolicyFromGroupError
);
export const selectUnlinkPolicyFromGroupError = createSelector(
    selectPoliciesState,
    (state) => state.unlinkPolicyFromGroupError
);

export const selectPolicyEvents = createSelector(selectPoliciesState, (state) => state.policyEvents);
export const selectTotalEvents = createSelector(selectPoliciesState, (state) => state.eventsTotal);
export const selectEventsPage = createSelector(selectPoliciesState, (state) => state.eventsPage);
export const selectEventsGridSearch = createSelector(selectPoliciesState, (state) => state.eventsGridSearch);
export const selectIsLoadingEvents = createSelector(selectPoliciesState, (state) => state.isLoadingEvents);
export const selectEventsGridFiltration = createSelector(selectPoliciesState, (state) => state.eventsGridFiltration);
export const selectEventsGridFiltrationByField = createSelector(selectEventsGridFiltration, (filtration) =>
    filtrationToDictionary(filtration)
);
export const selectEventsGridSorting = createSelector(selectPoliciesState, (state) => state.eventsSorting);

export const selectPolicyAgents = createSelector(selectPoliciesState, (state) => state.policyAgents);
export const selectTotalAgents = createSelector(selectPoliciesState, (state) => state.agentsTotal);
export const selectAgentsPage = createSelector(selectPoliciesState, (state) => state.agentsPage);
export const selectAgentsGridSearch = createSelector(selectPoliciesState, (state) => state.agentsGridSearch);
export const selectIsLoadingAgents = createSelector(selectPoliciesState, (state) => state.isLoadingAgents);
export const selectAgentsGridFiltration = createSelector(selectPoliciesState, (state) => state.agentsGridFiltration);
export const selectAgentsGridFiltrationByField = createSelector(selectAgentsGridFiltration, (filtration) =>
    filtrationToDictionary(filtration)
);
export const selectAgentsGridSorting = createSelector(selectPoliciesState, (state) => state.agentsSorting);
export const selectSelectedAgentId = createSelector(selectPoliciesState, (state) => state.selectedAgentId);
export const selectSelectedAgent = createSelector(
    selectPolicyAgents,
    selectSelectedAgentId,
    (agents, selectedAgentId) => agents.find((agent) => agent.id === selectedAgentId)
);

export const selectPolicyGroups = createSelector(selectPoliciesState, (state) => state.policyGroups);
export const selectTotalGroups = createSelector(selectPoliciesState, (state) => state.groupsTotal);
export const selectGroupsPage = createSelector(selectPoliciesState, (state) => state.groupsPage);
export const selectGroupsGridSearch = createSelector(selectPoliciesState, (state) => state.groupsGridSearch);
export const selectIsLoadingGroups = createSelector(selectPoliciesState, (state) => state.isLoadingGroups);
export const selectGroupsGridFiltration = createSelector(selectPoliciesState, (state) => state.groupsGridFiltration);
export const selectGroupsGridFiltrationByField = createSelector(selectGroupsGridFiltration, (filtration) =>
    filtrationToDictionary(filtration)
);
export const selectGroupsGridSorting = createSelector(selectPoliciesState, (state) => state.groupsSorting);
export const selectSelectedPolicyGroupId = createSelector(selectPoliciesState, (state) => state.selectedPolicyGroupId);
export const selectSelectedPolicyGroup = createSelector(
    selectPolicyGroups,
    selectSelectedPolicyGroupId,
    (groups, selectedGroupId) => groups.find((group) => group.id === selectedGroupId)
);
export const selectIsLoadingModules = createSelector(selectPoliciesState, (state) => state.isLoadingModules);
export const selectIsUpdatingAgentData = createSelector(selectPoliciesState, (state) => state.isUpdatingAgentData);

export const selectInitialListQuery = createSelector(
    selectSelectedFilter,
    selectSelectedGroupId,
    selectGridFiltration,
    selectGridSearch,
    selectGridSorting,
    (filter, groupId, gridFiltration, gridSearch, sorting) => {
        const filters = [
            ...(filter?.value || []),
            ...(groupId
                ? [
                      {
                          field: 'group_id',
                          value: [groupId]
                      }
                  ]
                : []),
            ...gridFiltration,
            ...(gridSearch ? [{ field: 'data', value: gridSearch }] : [])
        ];

        return { filters, sort: sorting || {} };
    }
);
export const selectIsUpgradingAgents = createSelector(selectPoliciesState, (state) => state.isUpgradingAgents);
export const selectIsCancelUpgradingAgent = createSelector(
    selectPoliciesState,
    (state) => state.isCancelUpgradingAgent
);

export const selectFilterItemsGroupIds = createSelector(selectPoliciesState, (state) => state.filterItemsGroupIds);
export const selectFilterItemsModuleNames = createSelector(
    selectPoliciesState,
    (state) => state.filterItemsModuleNames
);
export const selectFilterItemsTags = createSelector(selectPoliciesState, (state) => state.filterItemsTags);

export const selectAgentFilterItemGroupIds = createSelector(
    selectPoliciesState,
    (state) => state.agentFilterItemGroupIds
);
export const selectAgentFilterItemModuleNames = createSelector(
    selectPoliciesState,
    (state) => state.agentFilterItemModuleNames
);
export const selectAgentFilterItemOs = createSelector(selectPoliciesState, (state) => state.agentFilterItemOs);
export const selectAgentFilterItemTags = createSelector(selectPoliciesState, (state) => state.agentFilterItemTags);

export const selectEventFilterItemModuleIds = createSelector(
    selectPoliciesState,
    (state) => state.eventFilterItemModuleIds
);
export const selectEventFilterItemAgentIds = createSelector(
    selectPoliciesState,
    (state) => state.eventFilterItemAgentIds
);
export const selectEventFilterItemGroupIds = createSelector(
    selectPoliciesState,
    (state) => state.eventFilterItemGroupIds
);

export const selectGroupFilterItemModuleNames = createSelector(
    selectPoliciesState,
    (state) => state.groupFilterItemModuleNames
);
export const selectGroupFilterItemPolicyIds = createSelector(
    selectPoliciesState,
    (state) => state.groupFilterItemPolicyIds
);
export const selectGroupFilterItemTags = createSelector(selectPoliciesState, (state) => state.groupFilterItemTags);
