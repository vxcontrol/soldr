import { createAction, props } from '@ngrx/store';

import {
    ErrorResponse,
    ModelsPassword,
    PrivateAgents,
    PrivateBinaries,
    PrivateGroups,
    PrivateOptionsActions,
    PrivateOptionsEvents,
    PrivateOptionsFields,
    PrivateOptionsTags,
    PrivatePolicies,
    PrivateServices,
    PrivateSystemModules,
    PublicInfo
} from '@soldr/api';
import { Architecture, Package, OperationSystem } from '@soldr/shared';

export enum ActionType {
    ChangePassword = '[shared] Change password',
    ChangePasswordSuccess = '[shared] Change password - Success',
    ChangePasswordFailure = '[shared] Change password - Failure',

    FetchAllAgents = '[shared] Fetch all agents',
    FetchAllAgentsSuccess = '[shared] Fetch all agents - Success',
    FetchAllAgentsFailure = '[shared] Fetch all agents - Failure',

    FetchAllGroups = '[shared] Fetch all groups',
    FetchAllGroupsSuccess = '[shared] Fetch all groups - Success',
    FetchAllGroupsFailure = '[shared] Fetch all groups - Failure',

    FetchAllPolicies = '[shared] Fetch all policies',
    FetchAllPoliciesSuccess = '[shared] Fetch all policies - Success',
    FetchAllPoliciesFailure = '[shared] Fetch all policies - Failure',

    FetchAllServices = '[shared] Fetch all services',
    FetchAllServicesSuccess = '[shared] Fetch all services - Success',
    FetchAllServicesFailure = '[shared] Fetch all services - Failure',

    FetchAllModules = '[shared] Fetch all modules',
    FetchAllModulesSuccess = '[shared] Fetch all modules - Success',
    FetchAllModulesFailure = '[shared] Fetch all modules - Failure',

    ExportBinaryFile = '[shared] Export binary file',
    ExportBinaryFileSuccess = '[shared] Export binary file - Success',
    ExportBinaryFileFailure = '[shared] Export binary file - Failure',

    FetchInfo = '[shared] Fetch info',
    FetchInfoSuccess = '[shared] Fetch info - Success',
    FetchInfoFailure = '[shared] Fetch info - Failure',

    FetchLatestAgentBinary = '[shared] Fetch latest agent binary',
    FetchLatestAgentBinarySuccess = '[shared] Fetch latest agent binary - Success',
    FetchLatestAgentBinaryFailure = '[shared] Fetch latest agent binary - Failure',

    FetchAgentBinaries = '[shared] Fetch agent binary',
    FetchAgentBinariesSuccess = '[shared] Fetch agent binary - Success',
    FetchAgentBinariesFailure = '[shared] Fetch agent binary - Failure',

    Logout = '[shared] Logout',
    LogoutSuccess = '[shared] Logout - Success',
    LogoutFailure = '[shared] Logout - Failure',

    FetchOptionsActions = '[shared] Fetch options actions',
    FetchOptionsActionsSuccess = '[shared] Fetch options actions - Success',
    FetchOptionsActionsFailure = '[shared] Fetch options actions - Failure',

    FetchOptionsEvents = '[shared] Fetch options events',
    FetchOptionsEventsSuccess = '[shared] Fetch options events - Success',
    FetchOptionsEventsFailure = '[shared] Fetch options events - Failure',

    FetchOptionsFields = '[shared] Fetch options fields',
    FetchOptionsFieldsSuccess = '[shared] Fetch options fields - Success',
    FetchOptionsFieldsFailure = '[shared] Fetch options fields - Failure',

    FetchOptionsTags = '[shared] Fetch options tags',
    FetchOptionsTagsSuccess = '[shared] Fetch options tags - Success',
    FetchOptionsTagsFailure = '[shared] Fetch options tags - Failure',

    SetFilterByTags = '[shared] Set filter by tags',
    ResetFilterByTags = '[shared] Reset filter by tags',
    SetFilterBySearchValue = '[shared] Set filter by search value'
}

export const changePassword = createAction(ActionType.ChangePassword, props<{ data: ModelsPassword }>());
export const changePasswordSuccess = createAction(ActionType.ChangePasswordSuccess);
export const changePasswordFailure = createAction(ActionType.ChangePasswordFailure, props<{ error: ErrorResponse }>());

export const fetchAllAgents = createAction(ActionType.FetchAllAgents);
export const fetchAllAgentsSuccess = createAction(ActionType.FetchAllAgentsSuccess, props<{ data: PrivateAgents }>());
export const fetchAllAgentsFailure = createAction(ActionType.FetchAllAgentsFailure);

export const fetchAllGroups = createAction(ActionType.FetchAllGroups, props<{ silent: boolean }>());
export const fetchAllGroupsSuccess = createAction(ActionType.FetchAllGroupsSuccess, props<{ data: PrivateGroups }>());
export const fetchAllGroupsFailure = createAction(ActionType.FetchAllGroupsFailure);

export const fetchAllModules = createAction(ActionType.FetchAllModules);
export const fetchAllModulesSuccess = createAction(
    ActionType.FetchAllModulesSuccess,
    props<{ data: PrivateSystemModules }>()
);
export const fetchAllModulesFailure = createAction(ActionType.FetchAllModulesFailure);

export const fetchAllPolicies = createAction(ActionType.FetchAllPolicies, props<{ silent: boolean }>());
export const fetchAllPoliciesSuccess = createAction(
    ActionType.FetchAllPoliciesSuccess,
    props<{ data: PrivatePolicies }>()
);
export const fetchAllPoliciesFailure = createAction(ActionType.FetchAllPoliciesFailure);

export const fetchAllServices = createAction(ActionType.FetchAllServices);
export const fetchAllServicesSuccess = createAction(
    ActionType.FetchAllServicesSuccess,
    props<{ data: PrivateServices }>()
);
export const fetchAllServicesFailure = createAction(
    ActionType.FetchAllServicesFailure,
    props<{ error: ErrorResponse }>()
);

export const exportBinaryFile = createAction(
    ActionType.ExportBinaryFile,
    props<{ os: OperationSystem; arch: Architecture; pack: Package, version: string }>()
);
export const exportBinaryFileSuccess = createAction(ActionType.ExportBinaryFileSuccess);
export const exportBinaryFileFailure = createAction(
    ActionType.ExportBinaryFileFailure,
    props<{ error: ErrorResponse }>()
);

export const fetchInfo = createAction(ActionType.FetchInfo, props<{ refreshCookie: boolean }>());
export const fetchInfoSuccess = createAction(ActionType.FetchInfoSuccess, props<{ info: PublicInfo }>());
export const fetchInfoFailure = createAction(ActionType.FetchInfoFailure, props<{ error: unknown }>());

export const fetchLatestAgentBinary = createAction(ActionType.FetchLatestAgentBinary);
export const fetchLatestAgentBinarySuccess = createAction(
    ActionType.FetchLatestAgentBinarySuccess,
    props<{ binaries: PrivateBinaries }>()
);
export const fetchLatestAgentBinaryFailure = createAction(ActionType.FetchLatestAgentBinaryFailure);

export const fetchAgentBinaries = createAction(ActionType.FetchAgentBinaries);
export const fetchAgentBinariesSuccess = createAction(
    ActionType.FetchAgentBinariesSuccess,
    props<{ binaries: PrivateBinaries }>()
);
export const fetchAgentBinariesFailure = createAction(ActionType.FetchAgentBinariesFailure);

export const logout = createAction(ActionType.Logout);
export const logoutSuccess = createAction(ActionType.LogoutSuccess);
export const logoutFailure = createAction(ActionType.LogoutFailure);

export const fetchOptionsActions = createAction(ActionType.FetchOptionsActions);
export const fetchOptionsActionsSuccess = createAction(
    ActionType.FetchOptionsActionsSuccess,
    props<{ data: PrivateOptionsActions }>()
);
export const fetchOptionsActionsFailure = createAction(
    ActionType.FetchOptionsActionsFailure,
    props<{ error: ErrorResponse }>()
);

export const fetchOptionsEvents = createAction(ActionType.FetchOptionsEvents);
export const fetchOptionsEventsSuccess = createAction(
    ActionType.FetchOptionsEventsSuccess,
    props<{ data: PrivateOptionsEvents }>()
);
export const fetchOptionsEventsFailure = createAction(
    ActionType.FetchOptionsEventsFailure,
    props<{ error: ErrorResponse }>()
);

export const fetchOptionsFields = createAction(ActionType.FetchOptionsFields);
export const fetchOptionsFieldsSuccess = createAction(
    ActionType.FetchOptionsFieldsSuccess,
    props<{ data: PrivateOptionsFields }>()
);
export const fetchOptionsFieldsFailure = createAction(
    ActionType.FetchOptionsFieldsFailure,
    props<{ error: ErrorResponse }>()
);

export const fetchOptionsTags = createAction(ActionType.FetchOptionsTags);
export const fetchOptionsTagsSuccess = createAction(
    ActionType.FetchOptionsTagsSuccess,
    props<{ data: PrivateOptionsTags }>()
);
export const fetchOptionsTagsFailure = createAction(
    ActionType.FetchOptionsTagsFailure,
    props<{ error: ErrorResponse }>()
);

export const setFilterByTags = createAction(ActionType.SetFilterByTags, props<{ tags: string[] }>());
export const resetFilterByTags = createAction(ActionType.ResetFilterByTags);
export const setFilterBySearchValue = createAction(ActionType.SetFilterBySearchValue, props<{ searchValue: string }>());
