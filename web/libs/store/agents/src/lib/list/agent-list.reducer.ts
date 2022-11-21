import { createReducer, on } from '@ngrx/store';

import { ErrorResponse } from '@soldr/api';
import { Agent, manyAgentsToModels, oneAgentToModel, AgentModule, privateAgentModulesToModels } from '@soldr/models';
import { Filter, Filtration, Sorting } from '@soldr/shared';

import * as AgentListActions from './agent-list.actions';

export const agentListFeatureKey = 'agent-list';

export interface State {
    agent: Agent;
    agents: Agent[];
    deleteError: ErrorResponse;
    filterItemsGroupIds: string[];
    filterItemsModuleNames: string[];
    filterItemsOs: string[];
    filterItemsTags: string[];
    filterItemsVersions: string[];
    filters: Filter[];
    filtersCounters: Record<string, number>;
    gridFiltration: Filtration[];
    gridSearch: string;
    initialized: boolean;
    isBlockingAgents: boolean;
    isCancelUpgradingAgent: boolean;
    isDeletingAgents: boolean;
    isDeletingFromGroup: boolean;
    isInitializedAgent: boolean;
    isLoadingAgent: boolean;
    isLoadingAgentFilterItems: boolean;
    isLoadingAgents: boolean;
    isMovingAgents: boolean;
    isUpdatingAgentData: boolean;
    isUpdatingAgents: boolean;
    isUpgradingAgents: boolean;
    moveToGroupError: ErrorResponse;
    page: number;
    restored: boolean;
    selectedAgentModules: AgentModule[];
    selectedFilterId: string | undefined;
    selectedGroupId: string | undefined;
    selectedIds: number[];
    sorting: Sorting | Record<never, any>;
    total: number;
    updateError: ErrorResponse;
    versions: string[];
}

export const initialState: State = {
    agent: undefined,
    agents: [],
    deleteError: undefined,
    filterItemsGroupIds: [],
    filterItemsModuleNames: [],
    filterItemsOs: [],
    filterItemsTags: [],
    filterItemsVersions: [],
    filters: [],
    filtersCounters: {},
    gridFiltration: [],
    gridSearch: '',
    initialized: false,
    isBlockingAgents: false,
    isCancelUpgradingAgent: false,
    isDeletingAgents: false,
    isDeletingFromGroup: false,
    isInitializedAgent: false,
    isLoadingAgent: false,
    isLoadingAgentFilterItems: false,
    isLoadingAgents: false,
    isMovingAgents: false,
    isUpdatingAgentData: false,
    isUpdatingAgents: false,
    isUpgradingAgents: false,
    moveToGroupError: undefined,
    page: 0,
    restored: false,
    selectedAgentModules: [],
    selectedFilterId: undefined,
    selectedGroupId: undefined,
    selectedIds: [],
    sorting: {},
    total: 0,
    updateError: undefined,
    versions: []
};

export const reducer = createReducer(
    initialState,

    on(AgentListActions.updateFilters, (state, { filters }) => ({ ...state, filters, initialized: true })),
    on(AgentListActions.fetchCountersByFiltersSuccess, (state, { counters }) => ({
        ...state,
        filtersCounters: counters
    })),

    on(AgentListActions.fetchAgentsPage, (state) => ({ ...state, isLoadingAgents: true })),
    on(AgentListActions.fetchAgentsPageSuccess, (state, { data, page }) => ({
        ...state,
        isLoadingAgents: false,
        isInitializedAgent: true,
        agents: page === 1 ? manyAgentsToModels(data) : [...state.agents, ...manyAgentsToModels(data)],
        page: data.agents?.length > 0 ? page : state.page,
        total: data.total
    })),
    on(AgentListActions.fetchAgentsFailure, (state) => ({ ...state, isLoadingAgents: false })),

    on(AgentListActions.fetchFilterItems, (state) => ({ ...state, isLoadingAgentFilterItems: true })),
    on(AgentListActions.fetchFilterItemsSuccess, (state, { groupIds, moduleNames, versions, os, tags }) => ({
        ...state,
        isLoadingAgentFilterItems: false,
        filterItemsGroupIds: groupIds,
        filterItemsModuleNames: moduleNames,
        filterItemsVersions: versions,
        filterItemsOs: os,
        filterItemsTags: tags
    })),
    on(AgentListActions.fetchFilterItemsFailure, (state) => ({ ...state, isLoadingAgentFilterItems: false })),

    on(AgentListActions.updateAgentData, (state) => ({ ...state, isUpdatingAgentData: true })),
    on(AgentListActions.updateAgentDataSuccess, (state, { data }) => {
        const updatedAgents = manyAgentsToModels(data);
        const updatedAgentsById: Record<string, Agent> = updatedAgents.reduce(
            (acc, current) => ({ ...acc, [current.id]: current }),
            {}
        );
        const updatedAgentsIds = Object.keys(updatedAgentsById);

        return {
            ...state,
            agents: [
                ...state.agents.map((item) =>
                    updatedAgentsIds.includes(item.id.toString()) ? updatedAgentsById[item.id] : item
                )
            ],
            isUpdatingAgentData: false
        };
    }),
    on(AgentListActions.updateAgentDataFailure, (state) => ({ ...state, isUpdatingAgentData: false })),

    on(AgentListActions.fetchAgentsVersionsSuccess, (state, { versions }) => ({ ...state, versions })),

    on(AgentListActions.selectAgents, (state, { agents }) => ({ ...state, selectedIds: agents.map(({ id }) => id) })),
    on(AgentListActions.selectAgentsByIds, (state, { ids }) => ({ ...state, selectedIds: ids })),

    on(AgentListActions.fetchAgent, (state) => ({ ...state, isLoadingAgent: true })),
    on(AgentListActions.fetchAgentSuccess, (state, { data, modules }) => ({
        ...state,
        agent: oneAgentToModel(data),
        selectedAgentModules: privateAgentModulesToModels(modules),
        isLoadingAgent: false
    })),
    on(AgentListActions.fetchAgentFailure, (state) => ({ ...state, isLoadingAgent: false })),

    on(AgentListActions.upgradeAgents, (state) => ({ ...state, isUpgradingAgents: true })),
    on(AgentListActions.upgradeAgentsSuccess, (state) => ({ ...state, isUpgradingAgents: false })),
    on(AgentListActions.upgradeAgentsFailure, (state) => ({ ...state, isUpgradingAgents: false })),

    on(AgentListActions.cancelUpgradeAgent, (state) => ({ ...state, isCancelUpgradingAgent: true })),
    on(AgentListActions.cancelUpgradeAgentSuccess, (state) => ({ ...state, isCancelUpgradingAgent: false })),
    on(AgentListActions.cancelUpgradeAgentFailure, (state) => ({ ...state, isCancelUpgradingAgent: false })),

    on(AgentListActions.selectFilter, (state, { id }) => ({
        ...state,
        selectedFilterId: id,
        selectedGroupId: undefined,
        gridFiltration: [],
        gridSearch: ''
    })),
    on(AgentListActions.selectGroup, (state, { id }) => ({
        ...state,
        selectedGroupId: id,
        selectedFilterId: undefined,
        gridFiltration: [],
        gridSearch: ''
    })),
    on(AgentListActions.setGridFiltration, (state, { filtration }) => {
        const needRemoveFiltration =
            Array.isArray(filtration.value) && filtration.value.length === 1 && !filtration.value[0];
        const updatedFiltration = state.gridFiltration.filter((item: Filtration) => item.field !== filtration.field);

        return { ...state, gridFiltration: [...updatedFiltration, ...(needRemoveFiltration ? [] : [filtration])] };
    }),
    on(AgentListActions.setGridFiltrationByTag, (state, { tag }) => {
        const updatedFiltration = state.gridFiltration.filter((item: Filtration) => item.field !== 'tags');

        return { ...state, gridFiltration: [...updatedFiltration, { field: 'tags', value: [tag] }] };
    }),
    on(AgentListActions.setGridSearch, (state, { value }) => ({ ...state, gridSearch: value })),
    on(AgentListActions.resetFiltration, (state) => ({ ...state, gridFiltration: [] })),
    on(AgentListActions.setGridSorting, (state, { sorting }) => ({ ...state, sorting })),

    on(AgentListActions.moveAgentsToGroup, (state, { groupId }) => ({
        ...state,
        isMovingAgents: true,
        moveToGroupError: undefined,
        isDeletingFromGroup: !groupId
    })),
    on(AgentListActions.moveAgentsToGroupSuccess, (state) => ({ ...state, isMovingAgents: false })),
    on(AgentListActions.moveAgentsToGroupFailure, (state, { error }) => ({
        ...state,
        isMovingAgents: false,
        moveToGroupError: error
    })),

    on(AgentListActions.moveAgentsToNewGroup, (state) => ({
        ...state,
        isMovingAgents: true,
        moveToGroupError: undefined
    })),
    on(AgentListActions.moveAgentsToNewGroupSuccess, (state) => ({ ...state, isMovingAgents: false })),
    on(AgentListActions.moveAgentsToNewGroupFailure, (state, { error }) => ({
        ...state,
        isMovingAgents: false,
        moveToGroupError: error
    })),

    on(AgentListActions.updateAgent, (state) => ({ ...state, isUpdatingAgents: true, updateError: undefined })),
    on(AgentListActions.updateAgentSuccess, (state) => ({ ...state, isUpdatingAgents: false })),
    on(AgentListActions.updateAgentFailure, (state, { error }) => ({
        ...state,
        isUpdatingAgents: false,
        updateError: error
    })),

    on(AgentListActions.blockAgents, (state) => ({ ...state, isBlockingAgents: true })),
    on(AgentListActions.blockAgentsSuccess, (state) => ({ ...state, isBlockingAgents: false })),
    on(AgentListActions.blockAgentsFailure, (state) => ({ ...state, isBlockingAgents: false })),

    on(AgentListActions.deleteAgents, (state) => ({ ...state, isDeletingAgents: true, deleteError: undefined })),
    on(AgentListActions.deleteAgentsSuccess, (state) => ({ ...state, isDeletingAgents: false })),
    on(AgentListActions.deleteAgentsFailure, (state, { error }) => ({
        ...state,
        isDeletingAgents: false,
        deleteError: error
    })),

    on(AgentListActions.restoreState, (state, { restoredState }) => ({ ...state, restored: true, ...restoredState })),

    on(AgentListActions.resetAgentErrors, (state) => ({
        ...state,
        deleteError: undefined,
        moveToGroupError: undefined,
        updateError: undefined
    })),
    on(AgentListActions.reset, () => ({ ...initialState }))
);
