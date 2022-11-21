import { createFeatureSelector, createSelector } from '@ngrx/store';

import { filtrationToDictionary } from '@soldr/shared';

import * as fromAgents from './agent-card.reducer';

export const selectAgentState = createFeatureSelector<fromAgents.State>(fromAgents.agentCardFeatureKey);

export const selectSelectedAgent = createSelector(selectAgentState, (state) => state.agent);
export const selectIsUpgradingAgent = createSelector(selectAgentState, (state) => state.isUpgradingAgent);
export const selectIsUpgradingCancelAgent = createSelector(selectAgentState, (state) => state.isCancelUpgradingAgent);
export const selectIsMovingAgent = createSelector(selectAgentState, (state) => state.isMovingAgent);
export const selectIsUpdatingAgent = createSelector(selectAgentState, (state) => state.isUpdatingAgent);
export const selectIsBlockingAgent = createSelector(selectAgentState, (state) => state.isBlockingAgent);
export const selectIsDeletingAgent = createSelector(selectAgentState, (state) => state.isDeletingAgent);
export const selectIsDeletingFromGroup = createSelector(selectAgentState, (state) => state.isDeletingFromGroup);
export const selectAgentModules = createSelector(selectAgentState, (state) => state.modulesOfAgentCard);
export const selectIsLoadingAgent = createSelector(selectAgentState, (state) => state.isLoadingAgent);
export const selectAgentEvents = createSelector(selectAgentState, (state) => state.eventsOfAgentCard);
export const selectTotalEvents = createSelector(selectAgentState, (state) => state.totalEvents);
export const selectEventsPage = createSelector(selectAgentState, (state) => state.eventsPage);
export const selectEventsGridSearch = createSelector(selectAgentState, (state) => state.eventsGridSearch);
export const selectIsLoadingEvents = createSelector(selectAgentState, (state) => state.isLoadingEvents);
export const selectEventsGridFiltration = createSelector(selectAgentState, (state) => state.eventsGridFiltration);
export const selectEventsGridFiltrationByField = createSelector(selectEventsGridFiltration, (filtration) =>
    filtrationToDictionary(filtration)
);
export const selectEventsGridSorting = createSelector(selectAgentState, (state) => state.eventsSorting);

export const selectDeleteError = createSelector(selectAgentState, (state) => state.deleteError);
export const selectMoveToGroupError = createSelector(selectAgentState, (state) => state.moveToGroupError);
export const selectUpdateError = createSelector(selectAgentState, (state) => state.updateError);

export const selectEventFilterItemModuleIds = createSelector(
    selectAgentState,
    (state) => state.eventFilterItemModuleIds
);
