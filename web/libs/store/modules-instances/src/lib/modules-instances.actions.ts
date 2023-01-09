import { createAction, props } from '@ngrx/store';

import { ErrorResponse, ModelsModuleA, PrivateEvents, PrivateSystemShortModules } from '@soldr/api';
import { Filtration, Sorting, ViewMode } from '@soldr/shared';

export enum ActionType {
    Init = '[modules-instances] Init',

    FetchEvents = '[modules-instances] Fetch events',
    FetchEventsSuccess = '[modules-instances] Fetch events success',
    FetchEventsFailure = '[modules-instances] Fetch events failure',

    FetchModuleEventsFilterItems = '[modules-instances] Fetch module event filter items',
    FetchModuleEventFiltersItemsSuccess = '[modules-instances] Fetch module events filter items success',
    FetchModuleEventFiltersItemsFailure = '[modules-instances] Fetch module events filter items failure',

    SetEventsGridSearch = '[modules-instances] Set events grid search',
    SetEventsGridFiltration = '[modules-instances] Set events grid filtration',
    ResetEventsFiltration = '[modules-instances] Reset events filtration',
    SetEventsGridSorting = '[modules-instances] Set events grid sorting',

    FetchModuleVersions = '[modules-instances] Fetch module versions',
    FetchModuleVersionsSuccess = '[modules-instances] Fetch module versions success',
    FetchModuleVersionsFailed = '[modules-instances] Fetch module versions failed',

    EnableModule = '[modules-instances] Enable module',
    EnableModuleSuccess = '[modules-instances] Enable module success',
    EnableModuleFailure = '[modules-instances] Enable module failure',

    DisableModule = '[modules-instances] Disable module',
    DisableModuleSuccess = '[modules-instances] Disable module success',
    DisableModuleFailure = '[modules-instances] Disable module failure',

    DeleteModule = '[modules-instances] Delete module',
    DeleteModuleSuccess = '[modules-instances] Delete module success',
    DeleteModuleFailure = '[modules-instances] Delete module failure',

    ChangeModuleVersion = '[modules-instances] Change module version ',
    ChangeModuleVersionSuccess = '[modules-instances] Change module version success',
    ChangeModuleVersionFailure = '[modules-instances] Change module version failure',

    UpdateModule = '[modules-instances] Update module',
    UpdateModuleSuccess = '[modules-instances] Update module success',
    UpdateModuleFailure = '[modules-instances] Update module failure',

    SaveModuleConfig = '[modules-instances] Save module',
    SaveModuleConfigSuccess = '[modules-instances] Save module success',
    SaveModuleConfigFailure = '[modules-instances] Save module failure',

    ResetModuleErrors = '[modules-instances] Reset module errors',
    Reset = '[modules-instances] Reset'
}

export const init = createAction(
    ActionType.Init,
    props<{ viewMode: ViewMode; entityId: number; moduleName: string }>()
);

export const fetchEvents = createAction(ActionType.FetchEvents, props<{ page?: number }>());
export const fetchEventsSuccess = createAction(
    ActionType.FetchEventsSuccess,
    props<{ data: PrivateEvents; page: number }>()
);
export const fetchEventsFailure = createAction(ActionType.FetchEventsFailure);

export const fetchModuleEventsFilterItems = createAction(ActionType.FetchModuleEventsFilterItems);
export const fetchModuleEventsFilterItemsSuccess = createAction(
    ActionType.FetchModuleEventFiltersItemsSuccess,
    props<{ agentIds: string[]; groupIds: string[] }>()
);
export const fetchModuleEventsFilterItemsFailure = createAction(
    ActionType.FetchModuleEventFiltersItemsFailure,
    props<{ error: ErrorResponse }>()
);

export const setEventsGridFiltration = createAction(
    ActionType.SetEventsGridFiltration,
    props<{ filtration: Filtration }>()
);
export const setEventsGridSearch = createAction(ActionType.SetEventsGridSearch, props<{ value: string }>());
export const resetEventsFiltration = createAction(ActionType.ResetEventsFiltration);
export const setEventsGridSorting = createAction(ActionType.SetEventsGridSorting, props<{ sorting: Sorting }>());

export const fetchModuleVersions = createAction(ActionType.FetchModuleVersions, props<{ moduleName: string }>());
export const fetchModuleVersionSuccess = createAction(
    ActionType.FetchModuleVersionsSuccess,
    props<{ data: PrivateSystemShortModules }>()
);
export const fetchModuleVersionsFailed = createAction(ActionType.FetchModuleVersionsFailed);

export const enableModule = createAction(ActionType.EnableModule, props<{ policyHash: string; moduleName: string }>());
export const enableModuleSuccess = createAction(ActionType.EnableModuleSuccess);
export const enableModuleFailure = createAction(ActionType.EnableModuleFailure, props<{ error: ErrorResponse }>());

export const disableModule = createAction(
    ActionType.DisableModule,
    props<{ policyHash: string; moduleName: string }>()
);
export const disableModuleSuccess = createAction(ActionType.DisableModuleSuccess);
export const disableModuleFailure = createAction(ActionType.DisableModuleFailure, props<{ error: ErrorResponse }>());

export const deleteModule = createAction(ActionType.DeleteModule, props<{ policyHash: string; moduleName: string }>());
export const deleteModuleSuccess = createAction(ActionType.DeleteModuleSuccess);
export const deleteModuleFailure = createAction(ActionType.DeleteModuleFailure, props<{ error: ErrorResponse }>());

export const changeModuleVersion = createAction(
    ActionType.ChangeModuleVersion,
    props<{ policyHash: string; moduleName: string; version: string }>()
);
export const changeModuleVersionSuccess = createAction(ActionType.ChangeModuleVersionSuccess);
export const changeModuleVersionFailure = createAction(
    ActionType.ChangeModuleVersionFailure,
    props<{ error: ErrorResponse }>()
);

export const updateModule = createAction(
    ActionType.UpdateModule,
    props<{ policyHash: string; moduleName: string; version: string }>()
);
export const updateModuleSuccess = createAction(ActionType.UpdateModuleSuccess);
export const updateModuleFailure = createAction(ActionType.UpdateModuleFailure, props<{ error: ErrorResponse }>());

export const saveModuleConfig = createAction(
    ActionType.SaveModuleConfig,
    props<{ policyHash: string; module: ModelsModuleA }>()
);
export const saveModuleConfigSuccess = createAction(ActionType.SaveModuleConfigSuccess);
export const saveModuleConfigFailure = createAction(
    ActionType.SaveModuleConfigFailure,
    props<{ error: ErrorResponse }>()
);

export const resetModuleErrors = createAction(ActionType.ResetModuleErrors);
export const reset = createAction(ActionType.Reset);
