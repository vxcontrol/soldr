import { createReducer, on } from '@ngrx/store';

import { ErrorResponse, ModelsGroup } from '@soldr/api';
import {
    Agent,
    Event,
    Group,
    GroupModule,
    Policy,
    manyAgentsToModels,
    privateEventsToModels,
    privateGroupModulesToModels,
    privateGroupsToModels,
    privateGroupToModel,
    privatePoliciesToModels
} from '@soldr/models';
import { Filtration, Sorting } from '@soldr/shared';

import * as GroupsActions from './groups.actions';

export const groupsFeatureKey = 'groups';

export interface State {
    agentFilterItemGroupIds: string[];
    agentFilterItemModuleNames: string[];
    agentFilterItemOs: string[];
    agentFilterItemTags: string[];
    agentsGridFiltration: Filtration[];
    agentsGridSearch: string;
    agentsGridSorting: Sorting | Record<never, any>;
    agentsPage: number;
    agentsTotal: number;
    copyError: ErrorResponse;
    createError: ErrorResponse;
    createdGroup: ModelsGroup;
    deleteError: ErrorResponse;
    eventFilterItemAgentIds: string[];
    eventFilterItemModuleIds: string[];
    eventFilterItemPolicyIds: string[];
    eventsGridFiltration: Filtration[];
    eventsGridSearch: string;
    eventsPage: number;
    eventsSorting: Sorting | Record<never, any>;
    eventsTotal: number;
    filterItemsModuleNames: string[];
    filterItemsPolicyIds: string[];
    filterItemsTags: string[];
    gridFiltration: Filtration[];
    gridSearch: string;
    group: Group;
    groupAgents: Agent[];
    groupEvents: Event[];
    groupPolicies: Policy[];
    groups: Group[];
    isCancelUpgradingAgent: boolean;
    isCopyingGroup: boolean;
    isCreatingGroup: boolean;
    isDeletingGroup: boolean;
    isLinkingGroup: boolean;
    isLoadingAgentFilterItems: boolean;
    isLoadingAgents: boolean;
    isLoadingEventFilterItems: boolean;
    isLoadingEvents: boolean;
    isLoadingFilterItems: boolean;
    isLoadingGroup: boolean;
    isLoadingGroups: boolean;
    isLoadingPolicies: boolean;
    isLoadingPolicyFilterItems: boolean;
    isUnlinkingGroup: boolean;
    isUpdatingAgentData: boolean;
    isUpdatingGroup: boolean;
    isUpgradingAgents: boolean;
    linkGroupToPolicyError: ErrorResponse;
    modulesOfGroup: GroupModule[];
    page: number;
    policiesGridFiltration: Filtration[];
    policiesGridSearch: string;
    policiesGridSorting: Sorting | Record<never, any>;
    policiesPage: number;
    policiesTotal: number;
    policyFilterItemModuleNames: string[];
    policyFilterItemTags: string[];
    restored: boolean;
    selectedAgentId: number | undefined;
    selectedGroupId: string | undefined;
    selectedIds: number[];
    selectedPolicyId: number | undefined;
    sorting: Sorting | Record<never, any>;
    total: number;
    unlinkGroupFromPolicyError: ErrorResponse;
    updateError: ErrorResponse;
}

export const initialState: State = {
    agentFilterItemGroupIds: [],
    agentFilterItemModuleNames: [],
    agentFilterItemOs: [],
    agentFilterItemTags: [],
    agentsGridFiltration: [],
    agentsGridSearch: '',
    agentsGridSorting: {},
    agentsPage: 0,
    agentsTotal: 0,
    copyError: undefined,
    createError: undefined,
    createdGroup: undefined,
    deleteError: undefined,
    eventFilterItemAgentIds: [],
    eventFilterItemModuleIds: [],
    eventFilterItemPolicyIds: [],
    eventsGridFiltration: [],
    eventsGridSearch: '',
    eventsPage: 0,
    eventsSorting: {},
    eventsTotal: 0,
    filterItemsModuleNames: [],
    filterItemsPolicyIds: [],
    filterItemsTags: [],
    gridFiltration: [],
    gridSearch: '',
    group: undefined,
    groupAgents: [],
    groupEvents: [],
    groupPolicies: [],
    groups: [],
    isCancelUpgradingAgent: false,
    isCopyingGroup: false,
    isCreatingGroup: false,
    isDeletingGroup: false,
    isLinkingGroup: false,
    isLoadingAgentFilterItems: false,
    isLoadingAgents: false,
    isLoadingEventFilterItems: false,
    isLoadingEvents: false,
    isLoadingFilterItems: false,
    isLoadingGroup: false,
    isLoadingGroups: false,
    isLoadingPolicies: false,
    isLoadingPolicyFilterItems: false,
    isUnlinkingGroup: false,
    isUpdatingAgentData: false,
    isUpdatingGroup: false,
    isUpgradingAgents: false,
    linkGroupToPolicyError: undefined,
    modulesOfGroup: [],
    page: 0,
    policiesGridFiltration: [],
    policiesGridSearch: '',
    policiesGridSorting: {},
    policiesPage: 0,
    policiesTotal: 0,
    policyFilterItemModuleNames: [],
    policyFilterItemTags: [],
    restored: false,
    selectedAgentId: undefined,
    selectedGroupId: undefined,
    selectedIds: [],
    selectedPolicyId: undefined,
    sorting: {},
    total: 0,
    unlinkGroupFromPolicyError: undefined,
    updateError: undefined
};

export const reducer = createReducer(
    initialState,

    on(GroupsActions.fetchGroupsPage, (state) => ({ ...state, isLoadingGroups: true })),
    on(GroupsActions.fetchGroupsPageSuccess, (state, { data, page }) => ({
        ...state,
        isLoadingGroups: false,
        total: data.total,
        groups: page === 1 ? privateGroupsToModels(data) : [...state.groups, ...privateGroupsToModels(data)],
        page: data.groups?.length > 0 ? page : state.page
    })),
    on(GroupsActions.fetchGroupsPageFailure, (state) => ({ ...state, isLoadingGroups: false })),

    on(GroupsActions.fetchGroupAgents, (state) => ({ ...state, isLoadingAgents: true })),
    on(GroupsActions.fetchGroupAgentsSuccess, (state, { data, page }) => ({
        ...state,
        isLoadingAgents: false,
        groupAgents: page === 1 ? manyAgentsToModels(data) : [...state.groupAgents, ...manyAgentsToModels(data)],
        agentsPage: data.agents?.length > 0 ? page : state.agentsPage,
        agentsTotal: data.total
    })),
    on(GroupsActions.fetchGroupAgentsFailure, (state) => ({ ...state, isLoadingAgents: false })),
    on(GroupsActions.setAgentsGridSearch, (state, { value }) => ({ ...state, agentsGridSearch: value })),
    on(GroupsActions.setAgentsGridFiltration, (state, { filtration }) => {
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
    on(GroupsActions.setAgentsGridSorting, (state, { sorting }) => ({ ...state, agentsGridSorting: sorting })),
    on(GroupsActions.selectAgent, (state, { id }) => ({ ...state, selectedAgentId: id })),
    on(GroupsActions.resetAgentsFiltration, (state) => ({ ...state, agentsGridFiltration: [] })),

    on(GroupsActions.updateAgentData, (state) => ({ ...state, isUpdatingAgentData: true })),
    on(GroupsActions.updateAgentDataSuccess, (state, { data }) => {
        const updatedAgent = manyAgentsToModels(data)[0];

        return {
            ...state,
            groupAgents: [...state.groupAgents.map((item) => (item.id === updatedAgent.id ? updatedAgent : item))],
            isUpdatingAgentData: false
        };
    }),
    on(GroupsActions.updateAgentDataFailure, (state) => ({ ...state, isUpdatingAgentData: false })),

    on(GroupsActions.fetchGroup, (state) => ({ ...state, isLoadingGroup: true })),
    on(GroupsActions.fetchGroupSuccess, (state, { data, modules }) => ({
        ...state,
        isLoadingGroup: false,
        group: privateGroupToModel(data),
        modulesOfGroup: privateGroupModulesToModels(modules)
    })),
    on(GroupsActions.fetchGroupFailure, (state) => ({ ...state, isLoadingGroup: false })),

    on(GroupsActions.fetchGroupEvents, (state) => ({ ...state, isLoadingEvents: true })),
    on(GroupsActions.fetchGroupEventsSuccess, (state, { data, page }) => ({
        ...state,
        isLoadingEvents: false,
        groupEvents: page === 1 ? privateEventsToModels(data) : [...state.groupEvents, ...privateEventsToModels(data)],
        eventsPage: data.events?.length > 0 ? page : state.eventsPage,
        eventsTotal: data.total
    })),
    on(GroupsActions.fetchGroupEventsFailure, (state) => ({ ...state, isLoadingEvents: false })),

    on(GroupsActions.fetchFilterItems, (state) => ({ ...state, isLoadingFilterItems: true })),
    on(GroupsActions.fetchFilterItemsSuccess, (state, { policyIds, moduleNames, tags }) => ({
        ...state,
        isLoadingFilterItems: false,
        filterItemsPolicyIds: policyIds,
        filterItemsModuleNames: moduleNames,
        filterItemsTags: tags
    })),
    on(GroupsActions.fetchFilterItemsFailure, (state) => ({ ...state, isLoadingFilterItems: false })),

    on(GroupsActions.setEventsGridFiltration, (state, { filtration }) => {
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
    on(GroupsActions.setEventsGridSearch, (state, { value }) => ({ ...state, eventsGridSearch: value })),
    on(GroupsActions.resetEventsFiltration, (state) => ({ ...state, eventsGridFiltration: [] })),
    on(GroupsActions.setEventsGridSorting, (state, { sorting }) => ({ ...state, eventsSorting: sorting })),

    on(GroupsActions.fetchGroupPolicies, (state) => ({ ...state, isLoadingPolicies: true })),
    on(GroupsActions.fetchGroupPoliciesSuccess, (state, { data, page }) => ({
        ...state,
        isLoadingPolicies: false,
        groupPolicies:
            page === 1 ? privatePoliciesToModels(data) : [...state.groupPolicies, ...privatePoliciesToModels(data)],
        policiesPage: data.policies?.length > 0 ? page : state.policiesPage,
        policiesTotal: data.total
    })),
    on(GroupsActions.fetchGroupPoliciesFailure, (state) => ({ ...state, isLoadingPolicies: false })),

    on(GroupsActions.fetchAgentFilterItems, (state: State) => ({ ...state, isLoadingAgentFilterItems: true })),
    on(GroupsActions.fetchAgentFilterItemsSuccess, (state: State, { groupIds, moduleNames, os, tags }) => ({
        ...state,
        isLoadingAgentFilterItems: false,
        agentFilterItemGroupIds: groupIds,
        agentFilterItemModuleNames: moduleNames,
        agentFilterItemOs: os,
        agentFilterItemTags: tags
    })),
    on(GroupsActions.fetchAgentFilterItemsFailure, (state: State) => ({
        ...state,
        isLoadingAgentFilterItems: false
    })),

    on(GroupsActions.fetchPolicyFilterItems, (state: State) => ({ ...state, isLoadingPolicyFilterItems: true })),
    on(GroupsActions.fetchPolicyFilterItemsSuccess, (state: State, { moduleNames, tags }) => ({
        ...state,
        isLoadingPolicyFilterItems: false,
        policyFilterItemModuleNames: moduleNames,
        policyFilterItemTags: tags
    })),
    on(GroupsActions.fetchPolicyFilterItemsFailure, (state: State) => ({
        ...state,
        isLoadingPolicyFilterItems: false
    })),

    on(GroupsActions.fetchEventFilterItems, (state: State) => ({ ...state, isLoadingEventFilterItems: true })),
    on(GroupsActions.fetchEventFilterItemsSuccess, (state: State, { moduleIds, agentIds, policyIds }) => ({
        ...state,
        isLoadingEventFilterItems: false,
        eventFilterItemModuleIds: moduleIds,
        eventFilterItemAgentIds: agentIds,
        eventFilterItemPolicyIds: policyIds
    })),
    on(GroupsActions.fetchEventFilterItemsFailure, (state: State) => ({ ...state, isLoadingEventFilterItems: false })),

    on(GroupsActions.setPoliciesGridFiltration, (state, { filtration }) => {
        const needRemoveFiltration =
            Array.isArray(filtration.value) && filtration.value.length === 1 && !filtration.value[0];
        const updatedFiltration = state.policiesGridFiltration.filter(
            (item: Filtration) => item.field !== filtration.field
        );

        return {
            ...state,
            policiesGridFiltration: [...updatedFiltration, ...(needRemoveFiltration ? [] : [filtration])]
        };
    }),
    on(GroupsActions.setPoliciesGridSorting, (state, { sorting }) => ({ ...state, policiesGridSorting: sorting })),
    on(GroupsActions.setPoliciesGridSearch, (state, { value }) => ({ ...state, policiesGridSearch: value })),
    on(GroupsActions.resetPoliciesFiltration, (state) => ({ ...state, policiesGridFiltration: [] })),
    on(GroupsActions.selectPolicy, (state, { id }) => ({ ...state, selectedPolicyId: id })),

    on(GroupsActions.createGroup, (state) => ({ ...state, isCreatingGroup: true, createError: undefined })),
    on(GroupsActions.createGroupSuccess, (state, { group }) => ({
        ...state,
        createdGroup: group,
        isCreatingGroup: false
    })),
    on(GroupsActions.createGroupFailure, (state, { error }) => ({
        ...state,
        isCreatingGroup: false,
        createdGroup: undefined,
        createError: error
    })),

    on(GroupsActions.updateGroup, (state) => ({ ...state, isUpdatingGroup: true, updateError: undefined })),
    on(GroupsActions.updateGroupSuccess, (state) => ({ ...state, isUpdatingGroup: false })),
    on(GroupsActions.updateGroupFailure, (state, { error }) => ({
        ...state,
        isUpdatingGroup: false,
        updateError: error
    })),

    on(GroupsActions.copyGroup, (state) => ({ ...state, isCopyingGroup: true, copyError: undefined })),
    on(GroupsActions.copyGroupSuccess, (state, { group }) => ({
        ...state,
        isCopyingGroup: false,
        createdGroup: group
    })),
    on(GroupsActions.copyGroupFailure, (state, { error }) => ({
        ...state,
        isCopyingGroup: false,
        createdGroup: undefined,
        copyError: error
    })),

    on(GroupsActions.deleteGroup, (state) => ({ ...state, isDeletingGroup: true, deleteError: undefined })),
    on(GroupsActions.deleteGroupSuccess, (state) => ({ ...state, isDeletingGroup: false })),
    on(GroupsActions.deleteGroupFailure, (state, { error }) => ({
        ...state,
        isDeletingGroup: false,
        deleteError: error
    })),

    on(GroupsActions.linkGroupToPolicy, (state) => ({
        ...state,
        isLinkingGroup: true,
        linkGroupToPolicyError: undefined
    })),
    on(GroupsActions.linkGroupToPolicySuccess, (state) => ({ ...state, isLinkingGroup: false })),
    on(GroupsActions.linkGroupToPolicyFailure, (state, { error }) => ({
        ...state,
        isLinkingGroup: false,
        linkGroupToPolicyError: error
    })),

    on(GroupsActions.unlinkGroupFromPolicy, (state) => ({
        ...state,
        isUnlinkingGroup: true,
        unlinkGroupFromPolicyError: undefined
    })),
    on(GroupsActions.unlinkGroupFromPolicySuccess, (state) => ({ ...state, isUnlinkingGroup: false })),
    on(GroupsActions.unlinkGroupFromPolicyFailure, (state, { error }) => ({
        ...state,
        isUnlinkingGroup: false,
        unlinkGroupFromPolicyError: error
    })),

    on(GroupsActions.setGridFiltration, (state, { filtration }) => {
        const updatedFiltration = state.gridFiltration.filter((item: Filtration) => item.field !== filtration.field);

        return { ...state, gridFiltration: [...updatedFiltration, filtration], page: 1 };
    }),
    on(GroupsActions.setGridFiltrationByTag, (state, { tag }) => {
        const updatedFiltration = state.gridFiltration.filter((item: Filtration) => item.field !== 'tags');

        return { ...state, gridFiltration: [...updatedFiltration, { field: 'tags', value: [tag] }], page: 1 };
    }),
    on(GroupsActions.setGridSearch, (state, { value }) => ({ ...state, gridSearch: value, page: 1 })),
    on(GroupsActions.resetFiltration, (state) => ({ ...state, gridFiltration: [], page: 1 })),
    on(GroupsActions.setGridSorting, (state, { sorting }) => ({ ...state, sorting, page: 1 })),

    on(GroupsActions.selectGroup, (state, { id }) => ({ ...state, selectedGroupId: id })),
    on(GroupsActions.selectGroups, (state, { groups }) => ({ ...state, selectedIds: groups.map(({ id }) => id) })),

    on(GroupsActions.upgradeAgents, (state: State) => ({ ...state, isUpgradingAgents: true })),
    on(GroupsActions.upgradeAgentsSuccess, (state: State) => ({ ...state, isUpgradingAgents: false })),
    on(GroupsActions.upgradeAgentsFailure, (state: State) => ({ ...state, isUpgradingAgents: false })),

    on(GroupsActions.cancelUpgradeAgent, (state: State) => ({ ...state, isCancelUpgradingAgent: true })),
    on(GroupsActions.cancelUpgradeAgentSuccess, (state: State) => ({ ...state, isCancelUpgradingAgent: false })),
    on(GroupsActions.cancelUpgradeAgentFailure, (state: State) => ({ ...state, isCancelUpgradingAgent: false })),

    on(GroupsActions.restoreState, (state, { restoredState }) => ({ ...state, restored: true, ...restoredState })),

    on(GroupsActions.resetCreatedGroup, (state) => ({ ...state, createdGroup: undefined })),
    on(GroupsActions.resetGroupErrors, (state) => ({
        ...state,
        copyError: undefined,
        createError: undefined,
        deleteError: undefined,
        linkGroupToPolicyError: undefined,
        unlinkGroupFromPolicyError: undefined,
        updateError: undefined
    })),
    on(GroupsActions.reset, () => ({ ...initialState }))
);
