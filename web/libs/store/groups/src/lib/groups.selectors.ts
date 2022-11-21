import { createFeatureSelector, createSelector } from '@ngrx/store';

import { filtrationToDictionary } from '@soldr/shared';

import * as fromGroups from './groups.reducer';

export const selectGroupsState = createFeatureSelector<fromGroups.State>(fromGroups.groupsFeatureKey);
export const selectGroups = createSelector(selectGroupsState, (state) => state.groups);

export const selectGroup = createSelector(selectGroupsState, (state) => state.group);
export const selectTotal = createSelector(selectGroupsState, (state) => state.total);
export const selectCopyError = createSelector(selectGroupsState, (state) => state.copyError);
export const selectCreateError = createSelector(selectGroupsState, (state) => state.createError);
export const selectCreatedGroup = createSelector(selectGroupsState, (state) => state.createdGroup);
export const selectDeleteError = createSelector(selectGroupsState, (state) => state.deleteError);
export const selectUpdateError = createSelector(selectGroupsState, (state) => state.updateError);
export const selectSelectedGroupId = createSelector(selectGroupsState, (state) => state.selectedGroupId);
export const selectSelectedGroup = createSelector(selectGroups, selectSelectedGroupId, (groups, groupId) =>
    groups.find((group) => group.id.toString() === groupId)
);
export const selectSelectedGroupsIds = createSelector(selectGroupsState, (state) => state.selectedIds);
export const selectSelectedGroups = createSelector(selectGroupsState, (state) =>
    state.groups.filter(({ id }) => state.selectedIds.includes(id))
);
export const selectRestored = createSelector(selectGroupsState, (state) => state.restored);
export const selectIsCopyingGroup = createSelector(selectGroupsState, (state) => state.isCopyingGroup);
export const selectIsCreatingGroup = createSelector(selectGroupsState, (state) => state.isCreatingGroup);
export const selectIsDeletingGroup = createSelector(selectGroupsState, (state) => state.isDeletingGroup);
export const selectIsLinkingGroup = createSelector(selectGroupsState, (state) => state.isLinkingGroup);
export const selectIsLoadingGroup = createSelector(selectGroupsState, (state) => state.isLoadingGroup);
export const selectIsLoadingGroups = createSelector(selectGroupsState, (state) => state.isLoadingGroups);
export const selectIsUnlinkingGroup = createSelector(selectGroupsState, (state) => state.isUnlinkingGroup);
export const selectIsUpdatingGroup = createSelector(selectGroupsState, (state) => state.isUpdatingGroup);
export const selectLinkGroupToPolicyError = createSelector(selectGroupsState, (state) => state.linkGroupToPolicyError);
export const selectUnlinkGroupFromPolicyError = createSelector(
    selectGroupsState,
    (state) => state.unlinkGroupFromPolicyError
);
export const selectGridFiltration = createSelector(selectGroupsState, (state) => state.gridFiltration);
export const selectGridFiltrationByField = createSelector(selectGridFiltration, (gridFiltration) =>
    gridFiltration
        .filter((item) => (Array.isArray(item.value) ? !!item.value[0] : !!item.value))
        .reduce((acc, filter) => ({ ...acc, [filter.field]: filter }), {})
);
export const selectGridSearch = createSelector(selectGroupsState, (state) => state.gridSearch);
export const selectGridSorting = createSelector(selectGroupsState, (state) => state.sorting);
export const selectPage = createSelector(selectGroupsState, (state) => state.page);
export const selectGroupModules = createSelector(selectGroupsState, (state) => state.modulesOfGroup);

export const selectTotalAgents = createSelector(selectGroupsState, (state) => state.agentsTotal);
export const selectGroupAgents = createSelector(selectGroupsState, (state) => state.groupAgents);
export const selectAgentsGridSearch = createSelector(selectGroupsState, (state) => state.agentsGridSearch);
export const selectAgentsGridFiltration = createSelector(selectGroupsState, (state) => state.agentsGridFiltration);
export const selectIsLoadingAgents = createSelector(selectGroupsState, (state) => state.isLoadingAgents);
export const selectAgentsGridFiltrationByField = createSelector(selectAgentsGridFiltration, (filtration) =>
    filtrationToDictionary(filtration)
);
export const selectAgentsGridSorting = createSelector(selectGroupsState, (state) => state.agentsGridSorting);
export const selectAgentsPage = createSelector(selectGroupsState, (state) => state.agentsPage);
export const selectSelectedAgentId = createSelector(selectGroupsState, (state) => state.selectedAgentId);
export const selectSelectedAgent = createSelector(selectGroupAgents, selectSelectedAgentId, (agents, selectedAgentId) =>
    agents.find((agent) => agent.id === selectedAgentId)
);

export const selectTotalEvents = createSelector(selectGroupsState, (state) => state.eventsTotal);
export const selectGroupEvents = createSelector(selectGroupsState, (state) => state.groupEvents);
export const selectIsLoadingEvents = createSelector(selectGroupsState, (state) => state.isLoadingEvents);
export const selectEventsGridSearch = createSelector(selectGroupsState, (state) => state.eventsGridSearch);
export const selectEventsPage = createSelector(selectGroupsState, (state) => state.eventsPage);
export const selectEventsGridFiltration = createSelector(selectGroupsState, (state) => state.eventsGridFiltration);
export const selectEventsGridFiltrationByField = createSelector(selectEventsGridFiltration, (filtration) =>
    filtrationToDictionary(filtration)
);
export const selectEventsGridSorting = createSelector(selectGroupsState, (state) => state.eventsSorting);

export const selectGroupPolicies = createSelector(selectGroupsState, (state) => state.groupPolicies);
export const selectTotalPolicies = createSelector(selectGroupsState, (state) => state.policiesTotal);
export const selectPoliciesPage = createSelector(selectGroupsState, (state) => state.policiesPage);
export const selectPoliciesGridSearch = createSelector(selectGroupsState, (state) => state.policiesGridSearch);
export const selectIsLoadingPolicies = createSelector(selectGroupsState, (state) => state.isLoadingPolicies);
export const selectPoliciesGridFiltration = createSelector(selectGroupsState, (state) => state.policiesGridFiltration);
export const selectPoliciesGridFiltrationByField = createSelector(selectPoliciesGridFiltration, (filtration) =>
    filtrationToDictionary(filtration)
);
export const selectSelectedPolicyId = createSelector(selectGroupsState, (state) => state.selectedPolicyId);
export const selectSelectedPolicy = createSelector(
    selectGroupPolicies,
    selectSelectedPolicyId,
    (policies, selectedPolicyId) => policies.find((policy) => policy.id === selectedPolicyId)
);
export const selectIsUpdatingAgentData = createSelector(selectGroupsState, (state) => state.isUpdatingAgentData);

export const selectInitialListQuery = createSelector(
    selectGridFiltration,
    selectGridSearch,
    selectGridSorting,
    (gridFiltration, gridSearch, sorting) => {
        const filters = [...gridFiltration, ...(gridSearch ? [{ field: 'data', value: gridSearch }] : [])];

        return { filters, sort: sorting || {} };
    }
);
export const selectIsUpgradingAgents = createSelector(selectGroupsState, (state) => state.isUpgradingAgents);
export const selectIsCancelUpgradingAgent = createSelector(selectGroupsState, (state) => state.isCancelUpgradingAgent);

export const selectFilterItemsPolicyIds = createSelector(selectGroupsState, (state) => state.filterItemsPolicyIds);
export const selectFilterItemsModuleNames = createSelector(selectGroupsState, (state) => state.filterItemsModuleNames);
export const selectFilterItemsTags = createSelector(selectGroupsState, (state) => state.filterItemsTags);

export const selectAgentFilterItemGroupIds = createSelector(
    selectGroupsState,
    (state) => state.agentFilterItemGroupIds
);
export const selectAgentFilterItemModuleNames = createSelector(
    selectGroupsState,
    (state) => state.agentFilterItemModuleNames
);
export const selectAgentFilterItemOs = createSelector(selectGroupsState, (state) => state.agentFilterItemOs);
export const selectAgentFilterItemTags = createSelector(selectGroupsState, (state) => state.agentFilterItemTags);

export const selectPolicyFilterItemModuleNames = createSelector(
    selectGroupsState,
    (state) => state.policyFilterItemModuleNames
);
export const selectPolicyFilterItemTags = createSelector(selectGroupsState, (state) => state.policyFilterItemTags);

export const selectEventFilterItemModuleIds = createSelector(
    selectGroupsState,
    (state) => state.eventFilterItemModuleIds
);
export const selectEventFilterItemAgentIds = createSelector(
    selectGroupsState,
    (state) => state.eventFilterItemAgentIds
);
export const selectEventFilterItemPolicyIds = createSelector(
    selectGroupsState,
    (state) => state.eventFilterItemPolicyIds
);
