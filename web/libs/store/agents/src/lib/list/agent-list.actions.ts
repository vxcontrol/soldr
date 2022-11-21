import { createAction, props } from '@ngrx/store';

import { ErrorResponse, PrivateAgent, PrivateAgentModules, PrivateAgents } from '@soldr/api';
import { Agent, AgentUpgradeTask } from '@soldr/models';
import { Filter, Filtration, Sorting } from '@soldr/shared';

import { State } from './agent-list.reducer';

export enum ActionType {
    UpdateFilters = '[agent-list] Update filters',

    SelectFilter = '[agent-list] Select filter',
    SelectGroup = '[agent-list] Select group',

    FetchCountersByFilters = '[agent-list] Fetch counter by filters',
    FetchCountersByFiltersSuccess = '[agent-list] Fetch counter by filters success',

    FetchAgentsPage = '[agent-list] Fetch agents page',
    FetchAgentsPageSuccess = '[agent-list] Fetch agents page success',
    FetchAgentsPageFailure = '[agent-list] Fetch agents page failure',

    FetchFilterItems = '[agent-list] Fetch filter items',
    FetchFilterItemsSuccess = '[agent-list] Fetch filter items success',
    FetchFilterItemsFailure = '[agent-list] Fetch filter items failure',

    UpdateAgentData = '[agent-list] Update agent data',
    UpdateAgentDataSuccess = '[agent-list] Update agent data success',
    UpdateAgentDataFailure = '[agent-list] Update agent data failure',

    FetchVersions = '[agent-list] Fetch agents versions',
    FetchVersionsSuccess = '[agent-list] Fetch agents versions success',

    SelectAgents = '[agent-list] Select agents',
    SelectAgentsByIds = '[agent-list] Select agents by ids',

    FetchAgent = '[agent-list] Fetch agent',
    FetchAgentSuccess = '[agent-list] Fetch agent success',
    FetchAgentFailure = '[agent-list] Fetch agent failure',

    UpgradeSelectedAgent = '[agent-list] Upgrade selected agents',
    UpgradeSelectedAgentSuccess = '[agent-list] Upgrade selected agents success',
    UpgradeSelectedAgentFailure = '[agent-list] Upgrade selected agents failure',

    CancelUpgradeAgent = '[agent-list] Cancel upgrade agent',
    CancelUpgradeAgentSuccess = '[agent-list] Cancel upgrade agent success',
    CancelUpgradeAgentFailure = '[agent-list] Cancel upgrade agent failure',

    MoveAgentsToNewGroup = '[agent-list] Move agents to new group',
    MoveAgentsToNewGroupSuccess = '[agent-list] Move agents to new group success',
    MoveAgentsToNewGroupFailure = '[agent-list] Move agents to new group failure',

    MoveAgentsToGroup = '[agent-list] Move agents to group',
    MoveAgentsToGroupSuccess = '[agent-list] Move agents to group success',
    MoveAgentsToGroupFailure = '[agent-list] Move agents to group failure',

    UpdateAgent = '[agent-list] Update agent',
    UpdateAgentSuccess = '[agent-list] Update agent success',
    UpdateAgentFailure = '[agent-list] Update agent failure',

    BlockAgents = '[agent-list] Block agents',
    BlockAgentsSuccess = '[agent-list] Block agents success',
    BlockAgentsFailure = '[agent-list] Block agents failure',

    DeleteAgents = '[agent-list] Delete agents',
    DeleteAgentsSuccess = '[agent-list] Delete agents success',
    DeleteAgentsFailure = '[agent-list] Delete agents failure',

    SetFiltrationGrid = '[agent-list] Set filtration grid',
    SetFiltrationGridByTag = '[agent-list] Set filtration grid by tag',
    SetSearchGrid = '[agent-list] Set search grid',
    ResetFiltration = '[agent-list] Reset filtration',
    SetGridSorting = '[agent-list] Set sorting grid',

    RestoreState = '[agent-list] Restore state',

    ResetAgentErrors = '[agent-list] reset agent errors',
    Reset = '[agent-list] reset'
}

export const updateFilters = createAction(ActionType.UpdateFilters, props<{ filters: Filter[] }>());

export const selectFilter = createAction(ActionType.SelectFilter, props<{ id: string | undefined }>());
export const selectGroup = createAction(ActionType.SelectGroup, props<{ id: string | undefined }>());

export const fetchCountersByFilters = createAction(ActionType.FetchCountersByFilters);
export const fetchCountersByFiltersSuccess = createAction(
    ActionType.FetchCountersByFiltersSuccess,
    props<{ counters: Record<string, number> }>()
);

export const fetchAgentsPage = createAction(ActionType.FetchAgentsPage, props<{ page?: number }>());
export const fetchAgentsPageSuccess = createAction(
    ActionType.FetchAgentsPageSuccess,
    props<{ data: PrivateAgents; page: number }>()
);
export const fetchAgentsFailure = createAction(ActionType.FetchAgentsPageFailure);

export const fetchFilterItems = createAction(ActionType.FetchFilterItems);
export const fetchFilterItemsSuccess = createAction(
    ActionType.FetchFilterItemsSuccess,
    props<{ versions: string[]; moduleNames: string[]; groupIds: string[]; os: string[]; tags: string[] }>()
);
export const fetchFilterItemsFailure = createAction(
    ActionType.FetchFilterItemsFailure,
    props<{ error: ErrorResponse }>()
);

export const updateAgentData = createAction(ActionType.UpdateAgentData, props<{ agents: Agent[] }>());
export const updateAgentDataSuccess = createAction(ActionType.UpdateAgentDataSuccess, props<{ data: PrivateAgents }>());
export const updateAgentDataFailure = createAction(
    ActionType.UpdateAgentDataFailure,
    props<{ error: ErrorResponse }>()
);

export const fetchAgentsVersions = createAction(ActionType.FetchVersions);
export const fetchAgentsVersionsSuccess = createAction(
    ActionType.FetchVersionsSuccess,
    props<{ versions: string[] }>()
);

export const selectAgents = createAction(ActionType.SelectAgents, props<{ agents: Agent[] }>());
export const selectAgentsByIds = createAction(ActionType.SelectAgentsByIds, props<{ ids: number[] }>());

export const fetchAgent = createAction(ActionType.FetchAgent, props<{ hash: string }>());
export const fetchAgentSuccess = createAction(
    ActionType.FetchAgentSuccess,
    props<{ data: PrivateAgent; modules: PrivateAgentModules }>()
);
export const fetchAgentFailure = createAction(ActionType.FetchAgentFailure);

export const upgradeAgents = createAction(
    ActionType.UpgradeSelectedAgent,
    props<{ agents: Agent[]; version: string }>()
);
export const upgradeAgentsSuccess = createAction(ActionType.UpgradeSelectedAgentSuccess);
export const upgradeAgentsFailure = createAction(
    ActionType.UpgradeSelectedAgentFailure,
    props<{ error: ErrorResponse }>()
);

export const cancelUpgradeAgent = createAction(
    ActionType.CancelUpgradeAgent,
    props<{ hash: string; task: AgentUpgradeTask }>()
);
export const cancelUpgradeAgentSuccess = createAction(ActionType.CancelUpgradeAgentSuccess);
export const cancelUpgradeAgentFailure = createAction(ActionType.CancelUpgradeAgentFailure);

export const moveAgentsToGroup = createAction(
    ActionType.MoveAgentsToGroup,
    props<{ ids: number[]; groupId: number }>()
);
export const moveAgentsToGroupSuccess = createAction(ActionType.MoveAgentsToGroupSuccess);
export const moveAgentsToGroupFailure = createAction(
    ActionType.MoveAgentsToGroupFailure,
    props<{ error: ErrorResponse }>()
);

export const moveAgentsToNewGroup = createAction(
    ActionType.MoveAgentsToNewGroup,
    props<{ ids: number[]; groupName: string }>()
);
export const moveAgentsToNewGroupSuccess = createAction(ActionType.MoveAgentsToNewGroupSuccess);
export const moveAgentsToNewGroupFailure = createAction(
    ActionType.MoveAgentsToNewGroupFailure,
    props<{ error: ErrorResponse }>()
);

export const updateAgent = createAction(ActionType.UpdateAgent, props<{ agent: Agent }>());
export const updateAgentSuccess = createAction(ActionType.UpdateAgentSuccess);
export const updateAgentFailure = createAction(ActionType.UpdateAgentFailure, props<{ error: ErrorResponse }>());

export const blockAgents = createAction(ActionType.BlockAgents, props<{ ids: number[] }>());
export const blockAgentsSuccess = createAction(ActionType.BlockAgentsSuccess);
export const blockAgentsFailure = createAction(ActionType.BlockAgentsFailure, props<{ error: ErrorResponse }>());

export const deleteAgents = createAction(ActionType.DeleteAgents, props<{ ids: number[] }>());
export const deleteAgentsSuccess = createAction(ActionType.DeleteAgentsSuccess);
export const deleteAgentsFailure = createAction(ActionType.DeleteAgentsFailure, props<{ error: ErrorResponse }>());

export const setGridFiltration = createAction(ActionType.SetFiltrationGrid, props<{ filtration: Filtration }>());
export const setGridFiltrationByTag = createAction(ActionType.SetFiltrationGridByTag, props<{ tag: string }>());
export const setGridSearch = createAction(ActionType.SetSearchGrid, props<{ value: string }>());
export const resetFiltration = createAction(ActionType.ResetFiltration);
export const setGridSorting = createAction(ActionType.SetGridSorting, props<{ sorting: Sorting }>());

export const restoreState = createAction(ActionType.RestoreState, props<{ restoredState: Partial<State> }>());

export const resetAgentErrors = createAction(ActionType.ResetAgentErrors);
export const reset = createAction(ActionType.Reset);
