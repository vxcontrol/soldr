import { createReducer, on } from '@ngrx/store';

import { ErrorResponse } from '@soldr/api';
import {
    Agent,
    oneAgentToModel,
    AgentModule,
    privateAgentModulesToModels,
    privateEventsToModels,
    Event
} from '@soldr/models';
import { Filtration, getGridFiltration, Sorting } from '@soldr/shared';

import * as AgentCardActions from './agent-card.actions';

export const agentCardFeatureKey = 'agent-card';

export interface State {
    agent: Agent;
    deleteError: ErrorResponse;
    eventFilterItemModuleIds: string[];
    eventsGridFiltration: Filtration[];
    eventsGridSearch: string;
    eventsOfAgentCard: Event[];
    eventsPage: number;
    eventsSorting: Sorting | Record<never, any>;
    isBlockingAgent: boolean;
    isCancelUpgradingAgent: boolean;
    isDeletingAgent: boolean;
    isDeletingFromGroup: boolean;
    isLoadingAgent: boolean;
    isLoadingEventFilterItems: boolean;
    isLoadingEvents: boolean;
    isMovingAgent: boolean;
    isUpdatingAgent: boolean;
    isUpgradingAgent: boolean;
    modulesOfAgentCard: AgentModule[];
    moveToGroupError: ErrorResponse;
    totalEvents: number;
    updateError: ErrorResponse;
}

export const initialState: State = {
    agent: undefined,
    deleteError: undefined,
    eventFilterItemModuleIds: [],
    eventsGridFiltration: [],
    eventsGridSearch: '',
    eventsOfAgentCard: [],
    eventsPage: 0,
    eventsSorting: {},
    isBlockingAgent: false,
    isCancelUpgradingAgent: false,
    isDeletingAgent: false,
    isDeletingFromGroup: false,
    isLoadingAgent: false,
    isLoadingEventFilterItems: false,
    isLoadingEvents: false,
    isMovingAgent: false,
    isUpdatingAgent: false,
    isUpgradingAgent: false,
    modulesOfAgentCard: [],
    moveToGroupError: undefined,
    totalEvents: 0,
    updateError: undefined
};

export const reducer = createReducer(
    initialState,

    on(AgentCardActions.fetchAgent, (state) => ({ ...state, isLoadingAgent: true })),
    on(AgentCardActions.fetchAgentSuccess, (state, { data, modules }) => ({
        ...state,
        agent: oneAgentToModel(data),
        modulesOfAgentCard: privateAgentModulesToModels(modules),
        isLoadingAgent: false
    })),
    on(AgentCardActions.fetchAgentFailure, (state) => ({ ...state, isLoadingAgent: false })),

    on(AgentCardActions.fetchAgentEvents, (state) => ({ ...state, isLoadingEvents: true })),
    on(AgentCardActions.fetchAgentEventsSuccess, (state, { data, page }) => ({
        ...state,
        isLoadingEvents: false,
        eventsOfAgentCard:
            page === 1 ? privateEventsToModels(data) : [...state.eventsOfAgentCard, ...privateEventsToModels(data)],
        eventsPage: data.events?.length > 0 ? page : state.eventsPage,
        totalEvents: data.total
    })),
    on(AgentCardActions.fetchAgentEventsFailure, (state) => ({ ...state, isLoadingEvents: false })),

    on(AgentCardActions.fetchEventFilterItems, (state) => ({ ...state, isLoadingEventFilterItems: true })),
    on(AgentCardActions.fetchEventFilterItemsSuccess, (state, { moduleIds }) => ({
        ...state,
        isLoadingEventFilterItems: false,
        eventFilterItemModuleIds: moduleIds
    })),
    on(AgentCardActions.fetchEventFilterItemsFailure, (state) => ({ ...state, isLoadingEventFilterItems: false })),

    on(AgentCardActions.setEventsGridFiltration, (state, { filtration }) => ({
        ...state,
        eventsGridFiltration: getGridFiltration(filtration, state.eventsGridFiltration)
    })),
    on(AgentCardActions.setEventsGridSearch, (state, { value }) => ({ ...state, eventsGridSearch: value })),
    on(AgentCardActions.resetEventsFiltration, (state) => ({ ...state, eventsGridFiltration: [] })),
    on(AgentCardActions.setEventsGridSorting, (state, { sorting }) => ({ ...state, eventsSorting: sorting })),

    on(AgentCardActions.upgradeAgent, (state) => ({ ...state, isUpgradingAgent: true })),
    on(AgentCardActions.upgradeAgentSuccess, (state) => ({ ...state, isUpgradingAgent: false })),
    on(AgentCardActions.upgradeAgentFailure, (state) => ({ ...state, isUpgradingAgent: false })),

    on(AgentCardActions.cancelUpgradeAgent, (state) => ({ ...state, isCancelUpgradingAgent: true })),
    on(AgentCardActions.cancelUpgradeAgentSuccess, (state) => ({ ...state, isCancelUpgradingAgent: false })),
    on(AgentCardActions.cancelUpgradeAgentFailure, (state) => ({ ...state, isCancelUpgradingAgent: false })),

    on(AgentCardActions.moveAgentToGroup, (state, { groupId }) => ({
        ...state,
        isMovingAgent: true,
        moveToGroupError: undefined,
        isDeletingFromGroup: !groupId
    })),
    on(AgentCardActions.moveAgentToGroupSuccess, (state) => ({ ...state, isMovingAgent: false })),
    on(AgentCardActions.moveAgentToGroupFailure, (state, { error }) => ({
        ...state,
        isMovingAgent: false,
        moveToGroupError: error
    })),

    on(AgentCardActions.moveAgentToNewGroup, (state) => ({
        ...state,
        isMovingAgent: true,
        moveToGroupError: undefined
    })),
    on(AgentCardActions.moveAgentToNewGroupSuccess, (state) => ({ ...state, isMovingAgent: false })),
    on(AgentCardActions.moveAgentToNewGroupFailure, (state, { error }) => ({
        ...state,
        isMovingAgent: false,
        moveToGroupError: error
    })),

    on(AgentCardActions.updateAgent, (state) => ({ ...state, isUpdatingAgent: true, updateError: undefined })),
    on(AgentCardActions.updateAgentSuccess, (state) => ({ ...state, isUpdatingAgent: false })),
    on(AgentCardActions.updateAgentFailure, (state, { error }) => ({
        ...state,
        isUpdatingAgent: false,
        updateError: error
    })),

    on(AgentCardActions.blockAgent, (state) => ({ ...state, isBlockingAgent: true })),
    on(AgentCardActions.blockAgentSuccess, (state) => ({ ...state, isBlockingAgent: false })),
    on(AgentCardActions.blockAgentFailure, (state) => ({ ...state, isBlockingAgent: false })),

    on(AgentCardActions.deleteAgent, (state) => ({ ...state, isDeletingAgent: true, deleteError: undefined })),
    on(AgentCardActions.deleteAgentSuccess, (state) => ({ ...state, isDeletingAgent: false })),
    on(AgentCardActions.deleteAgentFailure, (state, { error }) => ({
        ...state,
        isDeletingAgent: false,
        deleteError: error
    })),

    on(AgentCardActions.resetAgentErrors, (state) => ({
        ...state,
        deleteError: undefined,
        moveToGroupError: undefined,
        updateError: undefined
    })),
    on(AgentCardActions.reset, () => ({ ...initialState }))
);
