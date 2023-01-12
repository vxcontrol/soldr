import { createReducer, on } from '@ngrx/store';

import { ErrorResponse, ModelsPolicy, PrivatePolicyCountResponse } from '@soldr/api';
import {
    privatePoliciesToModels,
    Policy,
    privatePolicyToModel,
    PolicyModule,
    privatePoliciesModulesToModels,
    Event,
    privateEventsToModels,
    Agent,
    manyAgentsToModels,
    Group,
    privateGroupsToModels
} from '@soldr/models';
import { clone, Filter, Filtration, Sorting } from '@soldr/shared';

import * as PoliciesActions from './policies.actions';

export const policiesFeatureKey = 'policies';

export interface State {
    agentFilterItemGroupIds: string[];
    agentFilterItemModuleNames: string[];
    agentFilterItemOs: string[];
    agentFilterItemTags: string[];
    agentsGridFiltration: Filtration[];
    agentsGridSearch: string;
    agentsPage: number;
    agentsSorting: Sorting | Record<never, any>;
    agentsTotal: number;
    copyError: ErrorResponse;
    createError: ErrorResponse;
    createdPolicy: ModelsPolicy;
    deleteError: ErrorResponse;
    eventFilterItemAgentNames: string[];
    eventFilterItemGroupIds: string[];
    eventFilterItemModuleIds: string[];
    eventsGridFiltration: Filtration[];
    eventsGridSearch: string;
    eventsPage: number;
    eventsSorting: Sorting | Record<never, any>;
    eventsTotal: number;
    filterItemsGroupIds: string[];
    filterItemsModuleNames: string[];
    filterItemsTags: string[];
    filters: Filter[];
    filtersCounters: PrivatePolicyCountResponse;
    gridFiltration: Filtration[];
    gridSearch: string;
    groupFilterItemModuleNames: string[];
    groupFilterItemPolicyIds: string[];
    groupFilterItemTags: string[];
    groupsGridFiltration: Filtration[];
    groupsGridSearch: string;
    groupsPage: number;
    groupsSorting: Sorting | Record<never, any>;
    groupsTotal: number;
    initialized: boolean;
    isCancelUpgradingAgent: boolean;
    isCopyingPolicy: boolean;
    isCreatingPolicy: boolean;
    isDeletingPolicy: boolean;
    isLinkingPolicy: boolean;
    isLoadingAgentFilterItems: boolean;
    isLoadingAgents: boolean;
    isLoadingEventFilterItems: boolean;
    isLoadingEvents: boolean;
    isLoadingFilterItems: boolean;
    isLoadingGroupFilterItems: boolean;
    isLoadingGroups: boolean;
    isLoadingModules: boolean;
    isLoadingPolicies: boolean;
    isLoadingPolicy: boolean;
    isUnlinkingPolicy: boolean;
    isUpdatingAgentData: boolean;
    isUpdatingPolicy: boolean;
    isUpgradingAgents: boolean;
    linkPolicyFromGroupError: ErrorResponse;
    modulesOfPolicy: PolicyModule[];
    page: number;
    policies: Policy[];
    policy: Policy;
    policyAgents: Agent[];
    policyEvents: Event[];
    policyGroups: Group[];
    restored: boolean;
    selectedAgentId: number | undefined;
    selectedFilterId: string | undefined;
    selectedGroupId: string | undefined;
    selectedIds: number[];
    selectedPolicyGroupId: number | undefined;
    selectedPolicyId: string | undefined;
    sorting: Sorting | Record<never, any>;
    total: number;
    unlinkPolicyFromGroupError: ErrorResponse;
    updateError: ErrorResponse;
}

export const initialState: State = {
    agentFilterItemGroupIds: [],
    agentFilterItemModuleNames: [],
    agentFilterItemOs: [],
    agentFilterItemTags: [],
    agentsGridFiltration: [],
    agentsGridSearch: '',
    agentsPage: 0,
    agentsSorting: undefined,
    agentsTotal: 0,
    copyError: undefined,
    createError: undefined,
    createdPolicy: undefined,
    deleteError: undefined,
    eventFilterItemAgentNames: [],
    eventFilterItemGroupIds: [],
    eventFilterItemModuleIds: [],
    eventsGridFiltration: [],
    eventsGridSearch: '',
    eventsPage: 0,
    eventsSorting: {},
    eventsTotal: 0,
    filterItemsGroupIds: [],
    filterItemsModuleNames: [],
    filterItemsTags: [],
    filters: [],
    filtersCounters: {
        all: 0,
        without_groups: 0
    },
    gridFiltration: [],
    gridSearch: '',
    groupFilterItemModuleNames: [],
    groupFilterItemPolicyIds: [],
    groupFilterItemTags: [],
    groupsGridFiltration: [],
    groupsGridSearch: '',
    groupsPage: 0,
    groupsSorting: undefined,
    groupsTotal: 0,
    initialized: false,
    isCancelUpgradingAgent: false,
    isCopyingPolicy: false,
    isCreatingPolicy: false,
    isDeletingPolicy: false,
    isLinkingPolicy: false,
    isLoadingAgentFilterItems: false,
    isLoadingAgents: false,
    isLoadingEventFilterItems: false,
    isLoadingEvents: false,
    isLoadingFilterItems: false,
    isLoadingGroupFilterItems: false,
    isLoadingGroups: false,
    isLoadingModules: false,
    isLoadingPolicies: false,
    isLoadingPolicy: false,
    isUnlinkingPolicy: false,
    isUpdatingAgentData: false,
    isUpdatingPolicy: false,
    isUpgradingAgents: false,
    linkPolicyFromGroupError: undefined,
    modulesOfPolicy: [],
    page: 0,
    policies: [],
    policy: undefined,
    policyAgents: [],
    policyEvents: [],
    policyGroups: [],
    restored: false,
    selectedAgentId: undefined,
    selectedFilterId: undefined,
    selectedGroupId: undefined,
    selectedIds: [],
    selectedPolicyGroupId: undefined,
    selectedPolicyId: undefined,
    sorting: {},
    total: 0,
    unlinkPolicyFromGroupError: undefined,
    updateError: undefined
};

export const reducer = createReducer(
    initialState,

    on(PoliciesActions.updateFilters, (state, { filters }) => ({ ...state, filters, initialized: true })),
    on(PoliciesActions.fetchCountersByFiltersSuccess, (state, { counters }) => ({
        ...state,
        filtersCounters: counters
    })),

    on(PoliciesActions.fetchPoliciesPage, (state) => ({ ...state, isLoadingPolicies: true })),
    on(PoliciesActions.fetchPoliciesPageSuccess, (state, { data, page }) => ({
        ...state,
        isLoadingPolicies: false,
        total: data.total,
        policies: page === 1 ? privatePoliciesToModels(data) : [...state.policies, ...privatePoliciesToModels(data)],
        page: data.policies?.length > 0 ? page : state.page
    })),
    on(PoliciesActions.fetchPoliciesPageFailure, (state) => ({ ...state, isLoadingPolicies: false })),

    on(PoliciesActions.fetchPolicy, (state) => ({ ...state, isLoadingPolicy: true })),
    on(PoliciesActions.fetchPolicySuccess, (state, { data }) => ({
        ...state,
        isLoadingPolicy: false,
        policy: privatePolicyToModel(data)
    })),
    on(PoliciesActions.fetchPolicyFailure, (state) => ({ ...state, isLoadingPolicy: false })),

    on(PoliciesActions.fetchFilterItems, (state) => ({ ...state, isLoadingFilterItems: true })),
    on(PoliciesActions.fetchFilterItemsSuccess, (state, { moduleNames, groupIds, tags }) => ({
        ...state,
        isLoadingFilterItems: false,
        filterItemsModuleNames: moduleNames,
        filterItemsGroupIds: groupIds,
        filterItemsTags: tags
    })),
    on(PoliciesActions.fetchFilterItemsFailure, (state) => ({ ...state, isLoadingFilterItems: false })),

    on(PoliciesActions.fetchAgentFilterItems, (state: State) => ({ ...state, isLoadingAgentFilterItems: true })),
    on(PoliciesActions.fetchAgentFilterItemsSuccess, (state: State, { groupIds, moduleNames, os, tags }) => ({
        ...state,
        isLoadingAgentFilterItems: false,
        agentFilterItemGroupIds: groupIds,
        agentFilterItemModuleNames: moduleNames,
        agentFilterItemOs: os,
        agentFilterItemTags: tags
    })),
    on(PoliciesActions.fetchAgentFilterItemsFailure, (state: State) => ({
        ...state,
        isLoadingAgentFilterItems: false
    })),

    on(PoliciesActions.fetchEventFilterItems, (state: State) => ({ ...state, isLoadingEventFilterItems: true })),
    on(PoliciesActions.fetchEventFilterItemsSuccess, (state: State, { moduleIds, agentNames, groupIds }) => ({
        ...state,
        isLoadingEventFilterItems: false,
        eventFilterItemModuleIds: moduleIds,
        eventFilterItemAgentNames: agentNames,
        eventFilterItemGroupIds: groupIds
    })),
    on(PoliciesActions.fetchEventFilterItemsFailure, (state: State) => ({
        ...state,
        isLoadingEventFilterItems: false
    })),

    on(PoliciesActions.fetchGroupFilterItems, (state: State) => ({ ...state, isLoadingGroupFilterItems: true })),
    on(PoliciesActions.fetchGroupFilterItemsSuccess, (state: State, { moduleNames, policyIds, tags }) => ({
        ...state,
        isLoadingGroupsFilterItems: false,
        groupFilterItemModuleNames: moduleNames,
        groupFilterItemPolicyIds: policyIds,
        groupFilterItemTags: tags
    })),
    on(PoliciesActions.fetchGroupFilterItemsFailure, (state: State) => ({
        ...state,
        isLoadingGroupsFilterItems: false
    })),

    on(PoliciesActions.fetchPolicyModules, (state) => ({ ...state, isLoadingModules: true })),
    on(PoliciesActions.fetchPolicyModulesSuccess, (state, { data }) => ({
        ...state,
        isLoadingModules: false,
        modulesOfPolicy: privatePoliciesModulesToModels(data)
    })),
    on(PoliciesActions.fetchPolicyModulesFailure, (state) => ({ ...state, isLoadingModules: false })),

    on(PoliciesActions.fetchPolicyEvents, (state) => ({ ...state, isLoadingEvents: true })),
    on(PoliciesActions.fetchPolicyEventsSuccess, (state, { data, page }) => ({
        ...state,
        isLoadingEvents: false,
        policyEvents:
            page === 1 ? privateEventsToModels(data) : [...state.policyEvents, ...privateEventsToModels(data)],
        eventsPage: data.events?.length > 0 ? page : state.eventsPage,
        eventsTotal: data.total
    })),
    on(PoliciesActions.fetchPolicyEventsFailure, (state) => ({ ...state, isLoadingEvents: false })),

    on(PoliciesActions.setEventsGridFiltration, (state, { filtration }) => {
        const needRemoveFiltration =
            Array.isArray(filtration.value) && filtration.value.length === 1 && !filtration.value[0];
        const updatedFiltration = state.eventsGridFiltration.filter(
            (item: Filtration) => item.field !== filtration.field
        );

        return {
            ...state,
            eventsGridFiltration: [...updatedFiltration, ...(needRemoveFiltration ? [] : [filtration])]
        };
    }),
    on(PoliciesActions.setEventsGridSearch, (state, { value }) => ({ ...state, eventsGridSearch: value })),
    on(PoliciesActions.resetEventsFiltration, (state) => ({ ...state, eventsGridFiltration: [] })),
    on(PoliciesActions.setEventsGridSorting, (state, { sorting }) => ({ ...state, eventsSorting: sorting })),

    on(PoliciesActions.fetchPolicyAgents, (state) => ({ ...state, isLoadingAgents: true })),
    on(PoliciesActions.fetchPolicyAgentsSuccess, (state, { data, page }) => ({
        ...state,
        isLoadingAgents: false,
        policyAgents: page === 1 ? manyAgentsToModels(data) : [...state.policyAgents, ...manyAgentsToModels(data)],
        agentsPage: data.agents?.length > 0 ? page : state.agentsPage,
        agentsTotal: data.total
    })),
    on(PoliciesActions.fetchPolicyAgentsFailure, (state) => ({ ...state, isLoadingAgents: false })),

    on(PoliciesActions.setAgentsGridFiltration, (state, { filtration }) => {
        const needRemoveFiltration =
            Array.isArray(filtration.value) && filtration.value.length === 1 && !filtration.value[0];
        const updatedFiltration = state.agentsGridFiltration.filter(
            (item: Filtration) => item.field !== filtration.field
        );

        return {
            ...state,
            agentsGridFiltration: [...updatedFiltration, ...(needRemoveFiltration ? [] : [filtration])]
        };
    }),
    on(PoliciesActions.setAgentsGridSearch, (state, { value }) => ({ ...state, agentsGridSearch: value })),
    on(PoliciesActions.resetAgentsFiltration, (state) => ({ ...state, agentsGridFiltration: [] })),
    on(PoliciesActions.setAgentsGridSorting, (state, { sorting }) => ({ ...state, agentsSorting: sorting })),

    on(PoliciesActions.selectAgent, (state, { id }) => ({ ...state, selectedAgentId: id })),

    on(PoliciesActions.updateAgentData, (state) => ({ ...state, isUpdatingAgentData: true })),
    on(PoliciesActions.updateAgentDataSuccess, (state, { data }) => {
        const updatedAgent = manyAgentsToModels(data)[0];

        return {
            ...state,
            policyAgents: [...state.policyAgents.map((item) => (item.id === updatedAgent.id ? updatedAgent : item))],
            isUpdatingAgentData: false
        };
    }),
    on(PoliciesActions.updateAgentDataFailure, (state) => ({ ...state, isUpdatingAgentData: false })),

    on(PoliciesActions.fetchPolicyGroups, (state) => ({ ...state, isLoadingGroups: true })),
    on(PoliciesActions.fetchPolicyGroupsSuccess, (state, { data, page }) => ({
        ...state,
        isLoadingGroups: false,
        policyGroups:
            page === 1 ? privateGroupsToModels(data) : [...state.policyGroups, ...privateGroupsToModels(data)],
        groupsPage: data.groups?.length > 0 ? page : state.groupsPage,
        groupsTotal: data.total
    })),
    on(PoliciesActions.fetchPolicyGroupsFailure, (state) => ({ ...state, isLoadingGroups: false })),

    on(PoliciesActions.setGroupsGridFiltration, (state, { filtration }) => {
        const needRemoveFiltration =
            Array.isArray(filtration.value) && filtration.value.length === 1 && !filtration.value[0];
        const updatedFiltration = state.groupsGridFiltration.filter(
            (item: Filtration) => item.field !== filtration.field
        );

        return {
            ...state,
            groupsGridFiltration: [...updatedFiltration, ...(needRemoveFiltration ? [] : [filtration])]
        };
    }),
    on(PoliciesActions.setGroupsGridSearch, (state, { value }) => ({ ...state, groupsGridSearch: value })),
    on(PoliciesActions.resetGroupsFiltration, (state) => ({ ...state, groupsGridFiltration: [] })),
    on(PoliciesActions.setGroupsGridSorting, (state, { sorting }) => ({ ...state, groupsSorting: sorting })),

    on(PoliciesActions.selectPolicyGroup, (state, { id }) => ({ ...state, selectedPolicyGroupId: id })),

    on(PoliciesActions.createPolicy, (state) => ({ ...state, isCreatingPolicy: true, createError: undefined })),
    on(PoliciesActions.createPolicySuccess, (state, { policy }) => ({
        ...state,
        isCreatingPolicy: false,
        createdPolicy: policy
    })),
    on(PoliciesActions.createPolicyFailure, (state, { error }) => ({
        ...state,
        isCreatingPolicy: false,
        createdPolicy: undefined,
        createError: error
    })),

    on(PoliciesActions.updatePolicy, (state) => ({ ...state, isUpdatingPolicy: true, updateError: undefined })),
    on(PoliciesActions.updatePolicySuccess, (state) => ({ ...state, isUpdatingPolicy: false })),
    on(PoliciesActions.updatePolicyFailure, (state, { error }) => ({
        ...state,
        isUpdatingPolicy: false,
        updateError: error
    })),

    on(PoliciesActions.copyPolicy, (state) => ({ ...state, isCopyingPolicy: true, copyError: undefined })),
    on(PoliciesActions.copyPolicySuccess, (state, { policy }) => ({
        ...state,
        isCopyingPolicy: false,
        createdPolicy: policy
    })),
    on(PoliciesActions.copyPolicyFailure, (state, { error }) => ({
        ...state,
        isCopyingPolicy: false,
        createdPolicy: undefined,
        copyError: error
    })),

    on(PoliciesActions.deletePolicy, (state) => ({ ...state, isDeletingPolicy: true, deleteError: undefined })),
    on(PoliciesActions.deletePolicySuccess, (state) => ({ ...state, isDeletingPolicy: false })),
    on(PoliciesActions.deletePolicyFailure, (state, { error }) => ({
        ...state,
        isDeletingPolicy: false,
        deleteError: error
    })),

    on(PoliciesActions.linkPolicyToGroup, (state) => ({
        ...state,
        isLinkingPolicy: true,
        linkPolicyFromGroupError: undefined
    })),
    on(PoliciesActions.linkPolicyToGroupSuccess, (state) => ({ ...state, isLinkingPolicy: false })),
    on(PoliciesActions.linkPolicyToGroupFailure, (state, { error }) => ({
        ...state,
        isLinkingPolicy: false,
        linkPolicyFromGroupError: error
    })),

    on(PoliciesActions.unlinkPolicyFromGroup, (state) => ({
        ...state,
        isUnlinkingPolicy: true,
        unlinkPolicyFromGroupError: undefined
    })),
    on(PoliciesActions.unlinkPolicyFromGroupSuccess, (state) => ({ ...state, isUnlinkingPolicy: false })),
    on(PoliciesActions.unlinkPolicyFromGroupFailure, (state, { error }) => ({
        ...state,
        isUnlinkingPolicy: false,
        unlinkPolicyFromGroupError: error
    })),

    on(PoliciesActions.selectFilter, (state, { id }) => ({
        ...state,
        selectedFilterId: id,
        selectedGroupId: undefined,
        gridFiltration: [],
        gridSearch: ''
    })),
    on(PoliciesActions.selectGroup, (state, { id }) => ({
        ...state,
        selectedGroupId: id,
        selectedFilterId: undefined,
        gridFiltration: [],
        gridSearch: ''
    })),
    on(PoliciesActions.setGridFiltration, (state, { filtration }) => {
        const updatedFiltration = state.gridFiltration.filter((item: Filtration) => item.field !== filtration.field);

        return { ...state, gridFiltration: [...updatedFiltration, filtration], page: 1 };
    }),
    on(PoliciesActions.setGridFiltrationByTag, (state, { tag }) => {
        const updatedFiltration = state.gridFiltration.filter((item: Filtration) => item.field !== 'tags');

        return { ...state, gridFiltration: [...updatedFiltration, { field: 'tags', value: [tag] }], page: 1 };
    }),
    on(PoliciesActions.setGridSearch, (state, { value }) => ({ ...state, gridSearch: value, page: 1 })),
    on(PoliciesActions.resetFiltration, (state) => ({ ...state, gridFiltration: [], page: 1 })),
    on(PoliciesActions.setGridSorting, (state, { sorting }) => ({ ...state, sorting, page: 1 })),

    on(PoliciesActions.upgradeAgents, (state: State) => ({ ...state, isUpgradingAgents: true })),
    on(PoliciesActions.upgradeAgentsSuccess, (state: State) => ({ ...state, isUpgradingAgents: false })),
    on(PoliciesActions.upgradeAgentsFailure, (state: State) => ({ ...state, isUpgradingAgents: false })),

    on(PoliciesActions.cancelUpgradeAgent, (state: State) => ({ ...state, isCancelUpgradingAgent: true })),
    on(PoliciesActions.cancelUpgradeAgentSuccess, (state: State) => ({ ...state, isCancelUpgradingAgent: false })),
    on(PoliciesActions.cancelUpgradeAgentFailure, (state: State) => ({ ...state, isCancelUpgradingAgent: false })),

    on(PoliciesActions.selectPolicy, (state, { id }) => ({ ...state, selectedPolicyId: id })),
    on(PoliciesActions.selectPolicies, (state, { policies }) => ({
        ...state,
        selectedIds: policies.map(({ id }) => id)
    })),
    on(PoliciesActions.selectPoliciesByIds, (state, { ids }) => ({
        ...state,
        selectedIds: ids
    })),

    on(PoliciesActions.updatePolicyModuleConfig, (state, { module }) => {
        const modules = clone(state.modulesOfPolicy) as PolicyModule[];
        const updatedModules = modules.map((item) =>
            module.id === item.id
                ? {
                      ...module
                  }
                : item
        );

        return { ...state, modulesOfPolicy: updatedModules };
    }),

    on(PoliciesActions.restoreState, (state, { restoredState }) => ({ ...state, restored: true, ...restoredState })),

    on(PoliciesActions.resetCreatedPolicy, (state) => ({ ...state, createdPolicy: undefined })),
    on(PoliciesActions.resetPolicyErrors, (state) => ({
        ...state,
        copyError: undefined,
        createError: undefined,
        deleteError: undefined,
        linkPolicyFromGroupError: undefined,
        unlinkPolicyFromGroupError: undefined,
        updateError: undefined
    })),
    on(PoliciesActions.reset, () => ({ ...initialState }))
);
