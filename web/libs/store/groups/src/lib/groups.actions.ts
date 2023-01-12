import { createAction, props } from '@ngrx/store';

import {
    ErrorResponse,
    ModelsGroup,
    ModelsPolicy,
    PrivateAgents,
    PrivateEvents,
    PrivateGroup,
    PrivateGroupInfo,
    PrivateGroupModules,
    PrivateGroups,
    PrivatePolicies
} from '@soldr/api';
import { Agent, AgentUpgradeTask, Group } from '@soldr/models';
import { Filtration, Sorting } from '@soldr/shared';

import { State } from './groups.reducer';

export enum ActionType {
    FetchGroupsPage = '[groups] Fetch groups page',
    FetchGroupsPageSuccess = '[groups] Fetch groups page - Success',
    FetchGroupsPageFailure = '[groups] Fetch groups page - Failure',

    FetchGroup = '[groups] Fetch group',
    FetchGroupSuccess = '[groups] Fetch group - Success',
    FetchGroupFailure = '[groups] Fetch group - Failure',

    SelectGroup = '[group] Select group',
    SelectGroups = '[group] Select groups',

    CreateGroup = '[group] Create group',
    CreateGroupSuccess = '[group] Create group - Success',
    CreateGroupFailure = '[group] Create group - Failure',

    UpdateGroup = '[groups] Update group',
    UpdateGroupSuccess = '[groups] Update group - Success',
    UpdateGroupFailure = '[groups] Update group - Failure',

    CopyGroup = '[groups] Copy group',
    CopyGroupSuccess = '[groups] Copy group - Success',
    CopyGroupFailure = '[groups] Copy group - Failure',

    DeleteGroup = '[groups] Delete group',
    DeleteGroupSuccess = '[groups] Delete group - Success',
    DeleteGroupFailure = '[groups] Delete group - Failure',

    LinkGroupToPolicy = '[groups] Link group to policy',
    LinkGroupToPolicySuccess = '[groups] Link group to policy - Success',
    LinkGroupToPolicyFailure = '[groups] Link group to policy - Failure',

    UnlinkGroupToPolicy = '[groups] Unlink group from policy',
    UnlinkGroupToPolicySuccess = '[groups] Unlink group from policy - Success',
    UnlinkGroupToPolicyFailure = '[groups] Unlink group from policy - Failure',

    FetchGroupAgents = '[groups] Fetch group agents',
    FetchGroupAgentsSuccess = '[groups] Fetch group agents success',
    FetchGroupAgentsFailure = '[groups] Fetch group agents failure',

    SetAgentsGridSearch = '[groups] Set agents grid search',
    SetAgentsGridFiltration = '[groups] Set agents grid filtration',
    ResetAgentsFiltration = '[groups] Reset agents filtration',
    SetAgentsGridSorting = '[groups] Set agents grid sorting',
    SelectAgent = '[groups] Select agent',

    UpdateAgentData = '[groups] Update agent data',
    UpdateAgentDataSuccess = '[groups] Update agent data success',
    UpdateAgentDataFailure = '[groups] Update agent data failure',

    FetchGroupEvents = '[groups] Fetch group events',
    FetchGroupEventsSuccess = '[groups] Fetch group events success',
    FetchGroupEventsFailure = '[groups] Fetch group events failure',

    SetEventsGridSearch = '[groups] Set events grid search',
    SetEventsGridFiltration = '[groups] Set events grid filtration',
    ResetEventsFiltration = '[groups] Reset events filtration',
    SetEventsGridSorting = '[groups] Set events grid sorting',

    FetchGroupPolicies = '[groups] Fetch group policies',
    FetchGroupPoliciesSuccess = '[groups] Fetch group policies success',
    FetchGroupPoliciesFailure = '[groups] Fetch group policies failure',

    SetPoliciesGridFiltration = '[groups] Set policies grid search',
    SetPoliciesGridSearch = '[groups] Set policies grid filtration',
    ResetPoliciesFiltration = '[groups] Reset policies filtration',
    SetPoliciesGridSorting = '[groups] Set policies grid sorting',
    SelectPolicy = '[groups] Select policy',

    SetFiltrationGrid = '[groups] Set filtration grid',
    SetFiltrationGridByTag = '[groups] Set filtration grid by tag',
    SetSearchGrid = '[groups] Set search grid',
    ResetFiltration = '[groups] Reset filtration',
    SetGridSorting = '[groups] Set sorting grid',

    FetchAgentFilterItems = '[groups] Fetch agent filter items',
    FetchAgentFilterItemsSuccess = '[groups] Fetch agent filter items success',
    FetchAgentFilterItemsFailure = '[groups] Fetch agent filter items failure',

    FetchPolicyFilterItems = '[groups] Fetch policy filter items',
    FetchPolicyFilterItemsSuccess = '[groups] Fetch policy filter items success',
    FetchPolicyFilterItemsFailure = '[groups] Fetch policy filter items failure',

    FetchEventFilterItems = '[groups] Fetch event filter items',
    FetchEventFilterItemsSuccess = '[groups] Fetch event filter items success',
    FetchEventFilterItemsFailure = '[groups] Fetch event filter items failure',

    UpgradeSelectedAgent = '[groups] Upgrade selected agents',
    UpgradeSelectedAgentSuccess = '[groups] Upgrade selected agents success',
    UpgradeSelectedAgentFailure = '[groups] Upgrade selected agents failure',

    CancelUpgradeAgent = '[groups]  Cancel upgrade agent',
    CancelUpgradeAgentSuccess = '[groups]  Cancel upgrade agent success',
    CancelUpgradeAgentFailure = '[groups]  Cancel upgrade agent failure',

    RestoreState = '[groups] Restore state',

    FetchFilterItems = '[groups] Fetch filter items',
    FetchFilterItemsSuccess = '[groups] Fetch filter items success',
    FetchFilterItemsFailure = '[groups] Fetch filter items failure',

    ResetCreatedGroup = '[groups] reset created group',
    ResetGroupErrors = '[groups] reset group errors',

    Reset = '[groups] reset'
}

export const fetchGroupsPage = createAction(ActionType.FetchGroupsPage, props<{ page?: number }>());
export const fetchGroupsPageSuccess = createAction(
    ActionType.FetchGroupsPageSuccess,
    props<{ data: PrivateGroups; page: number }>()
);
export const fetchGroupsPageFailure = createAction(ActionType.FetchGroupsPageFailure);

export const fetchGroup = createAction(ActionType.FetchGroup, props<{ hash: string }>());
export const fetchGroupSuccess = createAction(
    ActionType.FetchGroupSuccess,
    props<{ data: PrivateGroup; modules: PrivateGroupModules }>()
);
export const fetchGroupFailure = createAction(ActionType.FetchGroupFailure);

export const selectGroup = createAction(ActionType.SelectGroup, props<{ id: string }>());
export const selectGroups = createAction(ActionType.SelectGroups, props<{ groups: Group[] }>());

export const createGroup = createAction(ActionType.CreateGroup, props<{ group: PrivateGroupInfo }>());
export const createGroupSuccess = createAction(ActionType.CreateGroupSuccess, props<{ group: ModelsGroup }>());
export const createGroupFailure = createAction(ActionType.CreateGroupFailure, props<{ error: ErrorResponse }>());

export const updateGroup = createAction(ActionType.UpdateGroup, props<{ group: Group }>());
export const updateGroupSuccess = createAction(ActionType.UpdateGroupSuccess);
export const updateGroupFailure = createAction(ActionType.UpdateGroupFailure, props<{ error: ErrorResponse }>());

export const copyGroup = createAction(ActionType.CopyGroup, props<{ group: Group; redirect?: boolean }>());
export const copyGroupSuccess = createAction(ActionType.CopyGroupSuccess, props<{ group: ModelsGroup }>());
export const copyGroupFailure = createAction(ActionType.CopyGroupFailure, props<{ error: ErrorResponse }>());

export const deleteGroup = createAction(ActionType.DeleteGroup, props<{ hash: string }>());
export const deleteGroupSuccess = createAction(ActionType.DeleteGroupSuccess);
export const deleteGroupFailure = createAction(ActionType.DeleteGroupFailure, props<{ error: ErrorResponse }>());

export const linkGroupToPolicy = createAction(
    ActionType.LinkGroupToPolicy,
    props<{ hash: string; policy: ModelsPolicy }>()
);
export const linkGroupToPolicySuccess = createAction(ActionType.LinkGroupToPolicySuccess);
export const linkGroupToPolicyFailure = createAction(
    ActionType.LinkGroupToPolicyFailure,
    props<{ error: ErrorResponse }>()
);

export const unlinkGroupFromPolicy = createAction(
    ActionType.UnlinkGroupToPolicy,
    props<{ hash: string; policy: ModelsPolicy }>()
);
export const unlinkGroupFromPolicySuccess = createAction(ActionType.UnlinkGroupToPolicySuccess);
export const unlinkGroupFromPolicyFailure = createAction(
    ActionType.UnlinkGroupToPolicyFailure,
    props<{ error: ErrorResponse }>()
);

export const fetchGroupAgents = createAction(ActionType.FetchGroupAgents, props<{ id: number; page?: number }>());
export const fetchGroupAgentsSuccess = createAction(
    ActionType.FetchGroupAgentsSuccess,
    props<{ data: PrivateAgents; page: number }>()
);
export const fetchGroupAgentsFailure = createAction(ActionType.FetchGroupAgentsFailure);
export const setAgentsGridSearch = createAction(ActionType.SetAgentsGridSearch, props<{ value: string }>());
export const setAgentsGridFiltration = createAction(
    ActionType.SetAgentsGridFiltration,
    props<{ filtration: Filtration }>()
);
export const resetAgentsFiltration = createAction(ActionType.ResetAgentsFiltration);
export const setAgentsGridSorting = createAction(ActionType.SetAgentsGridSorting, props<{ sorting: Sorting }>());
export const selectAgent = createAction(ActionType.SelectAgent, props<{ id: number }>());

export const updateAgentData = createAction(ActionType.UpdateAgentData, props<{ agent: Agent }>());
export const updateAgentDataSuccess = createAction(ActionType.UpdateAgentDataSuccess, props<{ data: PrivateAgents }>());
export const updateAgentDataFailure = createAction(ActionType.UpdateAgentDataFailure);

export const fetchGroupEvents = createAction(ActionType.FetchGroupEvents, props<{ id: number; page?: number }>());
export const fetchGroupEventsSuccess = createAction(
    ActionType.FetchGroupEventsSuccess,
    props<{ data: PrivateEvents; page: number }>()
);
export const fetchGroupEventsFailure = createAction(ActionType.FetchGroupEventsFailure);

export const fetchFilterItems = createAction(ActionType.FetchFilterItems);
export const fetchFilterItemsSuccess = createAction(
    ActionType.FetchFilterItemsSuccess,
    props<{ policyIds: string[]; moduleNames: string[]; tags: string[] }>()
);
export const fetchFilterItemsFailure = createAction(
    ActionType.FetchFilterItemsFailure,
    props<{ error: ErrorResponse }>()
);

export const setEventsGridFiltration = createAction(
    ActionType.SetEventsGridFiltration,
    props<{ filtration: Filtration }>()
);
export const setEventsGridSearch = createAction(ActionType.SetEventsGridSearch, props<{ value: string }>());
export const resetEventsFiltration = createAction(ActionType.ResetEventsFiltration);
export const setEventsGridSorting = createAction(ActionType.SetEventsGridSorting, props<{ sorting: Sorting }>());

export const fetchGroupPolicies = createAction(ActionType.FetchGroupPolicies, props<{ id: number; page?: number }>());
export const fetchGroupPoliciesSuccess = createAction(
    ActionType.FetchGroupPoliciesSuccess,
    props<{ data: PrivatePolicies; page: number }>()
);
export const fetchGroupPoliciesFailure = createAction(ActionType.FetchGroupPoliciesFailure);
export const setPoliciesGridFiltration = createAction(
    ActionType.SetPoliciesGridFiltration,
    props<{ filtration: Filtration }>()
);
export const setPoliciesGridSearch = createAction(ActionType.SetPoliciesGridSearch, props<{ value: string }>());
export const resetPoliciesFiltration = createAction(ActionType.ResetPoliciesFiltration);
export const setPoliciesGridSorting = createAction(ActionType.SetPoliciesGridSorting, props<{ sorting: Sorting }>());
export const selectPolicy = createAction(ActionType.SelectPolicy, props<{ id: number }>());

export const setGridFiltration = createAction(ActionType.SetFiltrationGrid, props<{ filtration: Filtration }>());
export const setGridFiltrationByTag = createAction(ActionType.SetFiltrationGridByTag, props<{ tag: string }>());
export const setGridSearch = createAction(ActionType.SetSearchGrid, props<{ value: string }>());
export const resetFiltration = createAction(ActionType.ResetFiltration);
export const setGridSorting = createAction(ActionType.SetGridSorting, props<{ sorting: Sorting }>());

export const fetchAgentFilterItems = createAction(ActionType.FetchAgentFilterItems);
export const fetchAgentFilterItemsSuccess = createAction(
    ActionType.FetchAgentFilterItemsSuccess,
    props<{ moduleNames: string[]; groupIds: string[]; os: string[]; tags: string[] }>()
);
export const fetchAgentFilterItemsFailure = createAction(
    ActionType.FetchAgentFilterItemsFailure,
    props<{ error: ErrorResponse }>()
);

export const fetchPolicyFilterItems = createAction(ActionType.FetchPolicyFilterItems);
export const fetchPolicyFilterItemsSuccess = createAction(
    ActionType.FetchPolicyFilterItemsSuccess,
    props<{ moduleNames: string[]; tags: string[] }>()
);
export const fetchPolicyFilterItemsFailure = createAction(
    ActionType.FetchPolicyFilterItemsFailure,
    props<{ error: ErrorResponse }>()
);

export const fetchEventFilterItems = createAction(ActionType.FetchEventFilterItems);
export const fetchEventFilterItemsSuccess = createAction(
    ActionType.FetchEventFilterItemsSuccess,
    props<{ moduleIds: string[]; agentNames: string[]; policyIds: string[] }>()
);
export const fetchEventFilterItemsFailure = createAction(
    ActionType.FetchEventFilterItemsFailure,
    props<{ error: ErrorResponse }>()
);

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

export const restoreState = createAction(ActionType.RestoreState, props<{ restoredState: Partial<State> }>());

export const resetCreatedGroup = createAction(ActionType.ResetCreatedGroup);
export const resetGroupErrors = createAction(ActionType.ResetGroupErrors);

export const reset = createAction(ActionType.Reset);
