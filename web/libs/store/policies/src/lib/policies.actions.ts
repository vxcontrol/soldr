import { createAction, props } from '@ngrx/store';

import {
    ErrorResponse,
    ModelsGroup,
    ModelsPolicy,
    PrivateAgents,
    PrivateEvents,
    PrivateGroups,
    PrivatePolicies,
    PrivatePolicy,
    PrivatePolicyInfo,
    PrivatePolicyModules
} from '@soldr/api';
import { Agent, AgentUpgradeTask, Policy } from '@soldr/models';
import { Filter, Filtration, Sorting } from '@soldr/shared';

import { State } from './policies.reducer';

export enum ActionType {
    UpdateFilters = '[policies] Update filters',

    SelectFilter = '[policies] Select filter',
    SelectGroup = '[policies] Select group',

    FetchCountersByFilters = '[policies] Fetch counter by filters',
    FetchCountersByFiltersSuccess = '[policies] Fetch counter by filters - Success',

    FetchAllPolicies = '[policies] Fetch all policies',
    FetchAllPoliciesSuccess = '[policies] Fetch all policies - Success',
    FetchAllPoliciesFailure = '[policies] Fetch all policies - Failure',

    FetchPoliciesPage = '[policies] Fetch policies page',
    FetchPoliciesPageSuccess = '[policies] Fetch policies page - Success',
    FetchPoliciesPageFailure = '[policies] Fetch policies page - Failure',

    FetchFilterItems = '[policies] Fetch filter items',
    FetchFilterItemsSuccess = '[policies] Fetch filter items success',
    FetchFilterItemsFailure = '[policies] Fetch filter items failure',

    FetchAgentFilterItems = '[policies] Fetch agent filter items',
    FetchAgentFilterItemsSuccess = '[policies] Fetch agent filter items success',
    FetchAgentFilterItemsFailure = '[policies] Fetch agent filter items failure',

    FetchEventFilterItems = '[policies] Fetch event filter items',
    FetchEventFilterItemsSuccess = '[policies] Fetch event filter items success',
    FetchEventFilterItemsFailure = '[policies] Fetch event filter items failure',

    FetchGroupFilterItems = '[policies] Fetch group filter items',
    FetchGroupFilterItemsSuccess = '[policies] Fetch group filter items success',
    FetchGroupFilterItemsFailure = '[policies] Fetch group filter items failure',

    FetchPolicy = '[policies] Fetch policy',
    FetchPolicySuccess = '[policies] Fetch policy - Success',
    FetchPolicyFailure = '[policies] Fetch policy - Failure',

    FetchPolicyModules = '[policies] Fetch policy modules',
    FetchPolicyModulesSuccess = '[policies] Fetch policy modules success',
    FetchPolicyModulesFailure = '[policies] Fetch policy modules failure',

    FetchPolicyEvents = '[policies] Fetch policy events',
    FetchPolicyEventsSuccess = '[policies] Fetch policy events success',
    FetchPolicyEventsFailure = '[policies] Fetch policy events failure',

    SetEventsGridSearch = '[policies] Set events grid search',
    SetEventsGridFiltration = '[policies] Set events grid filtration',
    ResetEventsFiltration = '[policies] Reset events filtration',
    SetEventsGridSorting = '[policies] Set events grid sorting',

    FetchPolicyAgents = '[policies] Fetch policy agents',
    FetchPolicyAgentsSuccess = '[policies] Fetch policy agents success',
    FetchPolicyAgentsFailure = '[policies] Fetch policy agents failure',

    SetAgentsGridSearch = '[policies] Set agents grid search',
    SetAgentsGridFiltration = '[policies] Set agents grid filtration',
    ResetAgentsFiltration = '[policies] Reset agents filtration',
    SetAgentsGridSorting = '[policies] Set agents grid sorting',

    SelectAgent = '[policies] Select agent',

    UpdateAgentData = '[policies] Update agent data',
    UpdateAgentDataSuccess = '[policies] Update agent data success',
    UpdateAgentDataFailure = '[policies] Update agent data failure',

    FetchPolicyGroups = '[policies] Fetch policy groups',
    FetchPolicyGroupsSuccess = '[policies] Fetch policy groups success',
    FetchPolicyGroupsFailure = '[policies] Fetch policy groups failure',

    SetGroupsGridSearch = '[policies] Set groups grid search',
    SetGroupsGridFiltration = '[policies] Set groups grid filtration',
    ResetGroupsFiltration = '[policies] Reset groups filtration',
    SetGroupsGridSorting = '[policies] Set groups grid sorting',

    SelectPolicyGroup = '[policies] Select group',

    SelectPolicy = '[policies] Select policy',
    SelectPolicies = '[policies] Select policies',
    SelectPoliciesByIds = '[policies] Select policies by ids',

    CreatePolicy = '[policies] Create policy',
    CreatePolicySuccess = '[policies] Create policy - Success',
    CreatePolicyFailure = '[policies] Create policy - Failure',

    UpdatePolicy = '[policies] Update policy',
    UpdatePolicySuccess = '[policies] Update policy - Success',
    UpdatePolicyFailure = '[policies] Update policy - Failure',

    CopyPolicy = '[policies] Copy policy',
    CopyPolicySuccess = '[policies] Copy policy - Success',
    CopyPolicyFailure = '[policies] Copy policy - Failure',

    DeletePolicy = '[policies] Delete policy',
    DeletePolicySuccess = '[policies] Delete policy - Success',
    DeletePolicyFailure = '[policies] Delete policy - Failure',

    LinkPolicyToGroup = '[policies] Link policy to group',
    LinkPolicyToGroupSuccess = '[policies] Link policy to group - Success',
    LinkPolicyToGroupFailure = '[policies] Link policy to group - Failure',

    UnlinkPolicyFromGroup = '[policies] Unlink policy from group',
    UnlinkPolicyFromGroupSuccess = '[policies] Unlink policy from group - Success',
    UnlinkPolicyFromGroupFailure = '[policies] Unlink policy from group - Failure',

    SetFiltrationGrid = '[policies] Set filtration grid',
    SetFiltrationGridByTag = '[policies] Set filtration grid by tag',
    SetSearchGrid = '[policies] Set search grid',
    ResetFiltration = '[policies] Reset filtration',
    SetGridSorting = '[policies] Set sorting grid',

    UpgradeSelectedAgent = '[groups] Upgrade selected agents',
    UpgradeSelectedAgentSuccess = '[groups] Upgrade selected agents success',
    UpgradeSelectedAgentFailure = '[groups] Upgrade selected agents failure',

    CancelUpgradeAgent = '[groups]  Cancel upgrade agent',
    CancelUpgradeAgentSuccess = '[groups]  Cancel upgrade agent success',
    CancelUpgradeAgentFailure = '[groups]  Cancel upgrade agent failure',

    RestoreState = '[policies] Restore state',

    ResetCreatedPolicy = '[policies] Reset created policy',
    ResetPolicyErrors = '[policies] Reset policy errors',

    Reset = '[policies] reset'
}

export const updateFilters = createAction(ActionType.UpdateFilters, props<{ filters: Filter[] }>());

export const selectFilter = createAction(ActionType.SelectFilter, props<{ id: string | undefined }>());
export const selectGroup = createAction(ActionType.SelectGroup, props<{ id: string | undefined }>());

export const fetchCountersByFilters = createAction(ActionType.FetchCountersByFilters);
export const fetchCountersByFiltersSuccess = createAction(
    ActionType.FetchCountersByFiltersSuccess,
    props<{ counters: Record<string, number> }>()
);

export const fetchAllPolicies = createAction(ActionType.FetchAllPolicies, props<{ silent: boolean }>());
export const fetchAllPoliciesSuccess = createAction(
    ActionType.FetchAllPoliciesSuccess,
    props<{ data: PrivatePolicies }>()
);
export const fetchAllPoliciesFailure = createAction(ActionType.FetchAllPoliciesFailure);

export const fetchPoliciesPage = createAction(ActionType.FetchPoliciesPage, props<{ page?: number }>());
export const fetchPoliciesPageSuccess = createAction(
    ActionType.FetchPoliciesPageSuccess,
    props<{ data: PrivatePolicies; page: number }>()
);
export const fetchPoliciesPageFailure = createAction(ActionType.FetchPoliciesPageFailure);

export const fetchFilterItems = createAction(ActionType.FetchFilterItems);
export const fetchFilterItemsSuccess = createAction(
    ActionType.FetchFilterItemsSuccess,
    props<{ moduleNames: string[]; groupIds: string[]; tags: string[] }>()
);
export const fetchFilterItemsFailure = createAction(
    ActionType.FetchFilterItemsFailure,
    props<{ error: ErrorResponse }>()
);

export const fetchAgentFilterItems = createAction(ActionType.FetchAgentFilterItems);
export const fetchAgentFilterItemsSuccess = createAction(
    ActionType.FetchAgentFilterItemsSuccess,
    props<{ moduleNames: string[]; groupIds: string[]; os: string[]; tags: string[] }>()
);
export const fetchAgentFilterItemsFailure = createAction(
    ActionType.FetchAgentFilterItemsFailure,
    props<{ error: ErrorResponse }>()
);

export const fetchEventFilterItems = createAction(ActionType.FetchEventFilterItems);
export const fetchEventFilterItemsSuccess = createAction(
    ActionType.FetchEventFilterItemsSuccess,
    props<{ moduleIds: string[]; agentIds: string[]; groupIds: string[] }>()
);
export const fetchEventFilterItemsFailure = createAction(
    ActionType.FetchEventFilterItemsFailure,
    props<{ error: ErrorResponse }>()
);

export const fetchGroupFilterItems = createAction(ActionType.FetchGroupFilterItems);
export const fetchGroupFilterItemsSuccess = createAction(
    ActionType.FetchGroupFilterItemsSuccess,
    props<{ policyIds: string[]; moduleNames: string[]; tags: string[] }>()
);
export const fetchGroupFilterItemsFailure = createAction(
    ActionType.FetchGroupFilterItemsFailure,
    props<{ error: ErrorResponse }>()
);

export const fetchPolicy = createAction(ActionType.FetchPolicy, props<{ hash: string }>());
export const fetchPolicySuccess = createAction(ActionType.FetchPolicySuccess, props<{ data: PrivatePolicy }>());
export const fetchPolicyFailure = createAction(ActionType.FetchPolicyFailure);

export const fetchPolicyModules = createAction(ActionType.FetchPolicyModules, props<{ hash: string }>());
export const fetchPolicyModulesSuccess = createAction(
    ActionType.FetchPolicyModulesSuccess,
    props<{ data: PrivatePolicyModules }>()
);
export const fetchPolicyModulesFailure = createAction(ActionType.FetchPolicyModulesFailure);

export const fetchPolicyEvents = createAction(ActionType.FetchPolicyEvents, props<{ id: number; page?: number }>());
export const fetchPolicyEventsSuccess = createAction(
    ActionType.FetchPolicyEventsSuccess,
    props<{ data: PrivateEvents; page: number }>()
);
export const fetchPolicyEventsFailure = createAction(ActionType.FetchPolicyEventsFailure);

export const setEventsGridFiltration = createAction(
    ActionType.SetEventsGridFiltration,
    props<{ filtration: Filtration }>()
);
export const setEventsGridSearch = createAction(ActionType.SetEventsGridSearch, props<{ value: string }>());
export const resetEventsFiltration = createAction(ActionType.ResetEventsFiltration);
export const setEventsGridSorting = createAction(ActionType.SetEventsGridSorting, props<{ sorting: Sorting }>());

export const fetchPolicyAgents = createAction(ActionType.FetchPolicyAgents, props<{ id: number; page?: number }>());
export const fetchPolicyAgentsSuccess = createAction(
    ActionType.FetchPolicyAgentsSuccess,
    props<{ data: PrivateAgents; page: number }>()
);
export const fetchPolicyAgentsFailure = createAction(ActionType.FetchPolicyAgentsFailure);

export const setAgentsGridFiltration = createAction(
    ActionType.SetAgentsGridFiltration,
    props<{ filtration: Filtration }>()
);
export const setAgentsGridSearch = createAction(ActionType.SetAgentsGridSearch, props<{ value: string }>());
export const resetAgentsFiltration = createAction(ActionType.ResetAgentsFiltration);
export const setAgentsGridSorting = createAction(ActionType.SetAgentsGridSorting, props<{ sorting: Sorting }>());

export const selectAgent = createAction(ActionType.SelectAgent, props<{ id: number }>());

export const updateAgentData = createAction(ActionType.UpdateAgentData, props<{ agent: Agent }>());
export const updateAgentDataSuccess = createAction(ActionType.UpdateAgentDataSuccess, props<{ data: PrivateAgents }>());
export const updateAgentDataFailure = createAction(ActionType.UpdateAgentDataFailure);

export const fetchPolicyGroups = createAction(ActionType.FetchPolicyGroups, props<{ id: number; page?: number }>());
export const fetchPolicyGroupsSuccess = createAction(
    ActionType.FetchPolicyGroupsSuccess,
    props<{ data: PrivateGroups; page: number }>()
);
export const fetchPolicyGroupsFailure = createAction(ActionType.FetchPolicyGroupsFailure);

export const setGroupsGridFiltration = createAction(
    ActionType.SetGroupsGridFiltration,
    props<{ filtration: Filtration }>()
);
export const setGroupsGridSearch = createAction(ActionType.SetGroupsGridSearch, props<{ value: string }>());
export const resetGroupsFiltration = createAction(ActionType.ResetGroupsFiltration);
export const setGroupsGridSorting = createAction(ActionType.SetGroupsGridSorting, props<{ sorting: Sorting }>());

export const selectPolicyGroup = createAction(ActionType.SelectPolicyGroup, props<{ id: number }>());

export const selectPolicy = createAction(ActionType.SelectPolicy, props<{ id: string }>());
export const selectPolicies = createAction(ActionType.SelectPolicies, props<{ policies: Policy[] }>());
export const selectPoliciesByIds = createAction(ActionType.SelectPoliciesByIds, props<{ ids: number[] }>());

export const createPolicy = createAction(ActionType.CreatePolicy, props<{ policy: PrivatePolicyInfo }>());
export const createPolicySuccess = createAction(ActionType.CreatePolicySuccess, props<{ policy: ModelsPolicy }>());
export const createPolicyFailure = createAction(ActionType.CreatePolicyFailure, props<{ error: ErrorResponse }>());

export const updatePolicy = createAction(ActionType.UpdatePolicy, props<{ policy: Policy }>());
export const updatePolicySuccess = createAction(ActionType.UpdatePolicySuccess);
export const updatePolicyFailure = createAction(ActionType.UpdatePolicyFailure, props<{ error: ErrorResponse }>());

export const copyPolicy = createAction(ActionType.CopyPolicy, props<{ policy: Policy; redirect?: boolean }>());
export const copyPolicySuccess = createAction(ActionType.CopyPolicySuccess, props<{ policy: ModelsPolicy }>());
export const copyPolicyFailure = createAction(ActionType.CopyPolicyFailure, props<{ error: ErrorResponse }>());

export const deletePolicy = createAction(ActionType.DeletePolicy, props<{ hash: string }>());
export const deletePolicySuccess = createAction(ActionType.DeletePolicySuccess);
export const deletePolicyFailure = createAction(ActionType.DeletePolicyFailure, props<{ error: ErrorResponse }>());

export const linkPolicyToGroup = createAction(
    ActionType.LinkPolicyToGroup,
    props<{ hash: string; group: ModelsGroup }>()
);
export const linkPolicyToGroupSuccess = createAction(ActionType.LinkPolicyToGroupSuccess);
export const linkPolicyToGroupFailure = createAction(
    ActionType.LinkPolicyToGroupFailure,
    props<{ error: ErrorResponse }>()
);

export const unlinkPolicyFromGroup = createAction(
    ActionType.UnlinkPolicyFromGroup,
    props<{ hash: string; group: ModelsGroup }>()
);
export const unlinkPolicyFromGroupSuccess = createAction(ActionType.UnlinkPolicyFromGroupSuccess);
export const unlinkPolicyFromGroupFailure = createAction(
    ActionType.UnlinkPolicyFromGroupFailure,
    props<{ error: ErrorResponse }>()
);

export const setGridFiltration = createAction(ActionType.SetFiltrationGrid, props<{ filtration: Filtration }>());
export const setGridFiltrationByTag = createAction(ActionType.SetFiltrationGridByTag, props<{ tag: string }>());
export const setGridSearch = createAction(ActionType.SetSearchGrid, props<{ value: string }>());
export const resetFiltration = createAction(ActionType.ResetFiltration);
export const setGridSorting = createAction(
    ActionType.SetGridSorting,
    props<{ sorting: Sorting | Record<never, any> }>()
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

export const resetCreatedPolicy = createAction(ActionType.ResetCreatedPolicy);
export const resetPolicyErrors = createAction(ActionType.ResetPolicyErrors);

export const reset = createAction(ActionType.Reset);
