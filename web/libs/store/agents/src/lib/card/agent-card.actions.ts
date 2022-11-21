import { createAction, props } from '@ngrx/store';

import { ErrorResponse, PrivateAgent, PrivateAgentModules, PrivateEvents } from '@soldr/api';
import { Agent, AgentUpgradeTask } from '@soldr/models';
import { Filtration, Sorting } from '@soldr/shared';

import { State } from './agent-card.reducer';

export enum ActionType {
    FetchAgent = '[agent-card] Fetch agent',
    FetchAgentSuccess = '[agent-card] Fetch agent success',
    FetchAgentFailure = '[agent-card] Fetch agent failure',

    FetchAgentEvents = '[agent-card] Fetch agent events',
    FetchAgentEventsSuccess = '[agent-card] Fetch agent events - Success',
    FetchAgentEventsFailure = '[agent-card] Fetch agent events - Failure',

    FetchEventFiltersItem = '[agent-card] Fetch filter items of events',
    FetchEventFiltersItemSuccess = '[agent-card] Fetch filter items of events - Success',
    FetchEventFiltersItemFailure = '[agent-card] Fetch filter items of events - Failure',

    SetEventsGridSearch = '[agent-card] Set events grid search',
    SetEventsGridFiltration = '[agent-card] Set events grid filtration',
    ResetEventsFiltration = '[agent-card] Reset events filtration',
    SetEventsGridSorting = '[agent-card] Set events grid sorting',

    UpgradeSelectedAgent = '[agent-card] Upgrade selected agent',
    UpgradeSelectedAgentSuccess = '[agent-card] Upgrade selected agent success',
    UpgradeSelectedAgentFailure = '[agent-card] Upgrade selected agent failure',

    CancelUpgradeAgent = '[agent-card] Cancel upgrade agent',
    CancelUpgradeAgentSuccess = '[agent-card] Cancel upgrade agent success',
    CancelUpgradeAgentFailure = '[agent-card] Cancel upgrade agent failure',

    MoveAgentToNewGroup = '[agent-card] Move agent to new group',
    MoveAgentToNewGroupSuccess = '[agent-card] Move agent to new group success',
    MoveAgentToNewGroupFailure = '[agent-card] Move agent to new group failure',

    MoveAgentToGroup = '[agent-card] Move agent to group',
    MoveAgentToGroupSuccess = '[agent-card] Move agent to group success',
    MoveAgentToGroupFailure = '[agent-card] Move agent to group failure',

    UpdateAgent = '[agent-card] Update agent',
    UpdateAgentSuccess = '[agent-card] Update agent success',
    UpdateAgentFailure = '[agent-card] Update agent failure',

    BlockAgent = '[agent-card] Block agent',
    BlockAgentSuccess = '[agent-card] Block agent success',
    BlockAgentFailure = '[agent-card] Block agent failure',

    DeleteAgent = '[agent-card] Delete agent',
    DeleteAgentSuccess = '[agent-card] Delete agent success',
    DeleteAgentFailure = '[agent-card] Delete agent failure',

    RestoreState = '[agent-card] Restore state',

    ResetAgentErrors = '[agent-card] reset agent errors',
    Reset = '[agent-card] reset'
}

export const fetchAgent = createAction(ActionType.FetchAgent, props<{ hash: string }>());
export const fetchAgentSuccess = createAction(
    ActionType.FetchAgentSuccess,
    props<{ data: PrivateAgent; modules: PrivateAgentModules }>()
);
export const fetchAgentFailure = createAction(ActionType.FetchAgentFailure);

export const fetchAgentEvents = createAction(ActionType.FetchAgentEvents, props<{ id: number; page?: number }>());
export const fetchAgentEventsSuccess = createAction(
    ActionType.FetchAgentEventsSuccess,
    props<{ data: PrivateEvents; page: number }>()
);
export const fetchAgentEventsFailure = createAction(ActionType.FetchAgentEventsFailure);

export const fetchEventFilterItems = createAction(ActionType.FetchEventFiltersItem);
export const fetchEventFilterItemsSuccess = createAction(
    ActionType.FetchEventFiltersItemSuccess,
    props<{ moduleIds: string[] }>()
);
export const fetchEventFilterItemsFailure = createAction(
    ActionType.FetchEventFiltersItemFailure,
    props<{ error: ErrorResponse }>()
);

export const setEventsGridFiltration = createAction(
    ActionType.SetEventsGridFiltration,
    props<{ filtration: Filtration }>()
);
export const setEventsGridSearch = createAction(ActionType.SetEventsGridSearch, props<{ value: string }>());
export const resetEventsFiltration = createAction(ActionType.ResetEventsFiltration);
export const setEventsGridSorting = createAction(ActionType.SetEventsGridSorting, props<{ sorting: Sorting }>());

export const upgradeAgent = createAction(ActionType.UpgradeSelectedAgent, props<{ agent: Agent; version: string }>());
export const upgradeAgentSuccess = createAction(ActionType.UpgradeSelectedAgentSuccess);
export const upgradeAgentFailure = createAction(
    ActionType.UpgradeSelectedAgentFailure,
    props<{ error: ErrorResponse }>()
);

export const cancelUpgradeAgent = createAction(
    ActionType.CancelUpgradeAgent,
    props<{ hash: string; task: AgentUpgradeTask }>()
);
export const cancelUpgradeAgentSuccess = createAction(ActionType.CancelUpgradeAgentSuccess);
export const cancelUpgradeAgentFailure = createAction(ActionType.CancelUpgradeAgentFailure);

export const moveAgentToGroup = createAction(ActionType.MoveAgentToGroup, props<{ id: number; groupId: number }>());
export const moveAgentToGroupSuccess = createAction(ActionType.MoveAgentToGroupSuccess);
export const moveAgentToGroupFailure = createAction(
    ActionType.MoveAgentToGroupFailure,
    props<{ error: ErrorResponse }>()
);

export const moveAgentToNewGroup = createAction(
    ActionType.MoveAgentToNewGroup,
    props<{ id: number; groupName: string }>()
);
export const moveAgentToNewGroupSuccess = createAction(ActionType.MoveAgentToNewGroupSuccess);
export const moveAgentToNewGroupFailure = createAction(
    ActionType.MoveAgentToNewGroupFailure,
    props<{ error: ErrorResponse }>()
);

export const updateAgent = createAction(ActionType.UpdateAgent, props<{ agent: Agent }>());
export const updateAgentSuccess = createAction(ActionType.UpdateAgentSuccess);
export const updateAgentFailure = createAction(ActionType.UpdateAgentFailure, props<{ error: ErrorResponse }>());

export const blockAgent = createAction(ActionType.BlockAgent, props<{ id: number }>());
export const blockAgentSuccess = createAction(ActionType.BlockAgentSuccess);
export const blockAgentFailure = createAction(ActionType.BlockAgentFailure, props<{ error: ErrorResponse }>());

export const deleteAgent = createAction(ActionType.DeleteAgent, props<{ id: number }>());
export const deleteAgentSuccess = createAction(ActionType.DeleteAgentSuccess);
export const deleteAgentFailure = createAction(ActionType.DeleteAgentFailure, props<{ error: ErrorResponse }>());

export const restoreState = createAction(ActionType.RestoreState, props<{ restoredState: Partial<State> }>());

export const resetAgentErrors = createAction(ActionType.ResetAgentErrors);
export const reset = createAction(ActionType.Reset);
