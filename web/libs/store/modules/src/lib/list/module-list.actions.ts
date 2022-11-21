import { createAction, props } from '@ngrx/store';

import {
    ErrorResponse,
    ModelsModuleInfo,
    ModelsModuleS,
    PrivateSystemModules,
    PrivateSystemShortModules
} from '@soldr/api';
import { UploadModule } from '@soldr/features/modules';
import { Module } from '@soldr/models';
import { Filtration, Sorting } from '@soldr/shared';

import { State } from './module-list.reducer';

export enum ActionType {
    FetchModules = '[module-list] Fetch modules',
    FetchModulesSuccess = '[module-list] Fetch modules success',
    FetchModulesFailure = '[module-list] Fetch modules failure',

    FetchModulesPage = '[module-list] Fetch modules page',
    FetchModulesPageSuccess = '[module-list] Fetch modules page success',
    FetchModulesPageFailure = '[module-list] Fetch modules page failure',

    FetchModulesTags = '[module-list] Fetch modules tags',
    FetchModulesTagsSuccess = '[module-list] Fetch modules tags success',
    FetchModulesTagsFailure = '[module-list] Fetch modules tags failure',

    SelectModules = '[module-list]  Select modules',
    SelectModulesByIds = '[module-list] Select modules by ids',

    SetFiltrationGrid = '[module-list] Set filtration grid',
    SetFiltrationGridByTag = '[module-list] Set filtration grid by tag',
    SetSearchGrid = '[module-list] Set search grid',
    ResetFiltration = '[module-list] Reset filtration',
    SetGridSorting = '[module-list] Set sorting grid',

    CreateModule = '[module-list] Create module',
    CreateModuleSuccess = '[module-list] Create module success',
    CreateModuleFailure = '[module-list] Create module failure',

    ImportModule = '[module-list] Import Module',
    ImportModuleSuccess = '[module-list] Import Module success',
    ImportModuleFailure = '[module-list] Import Module failure',

    DeleteModule = '[module-list] Delete module',
    DeleteModuleSuccess = '[module-list] Delete module success',
    DeleteModuleFailure = '[module-list] Delete module failure',

    DeleteModuleVersion = '[module-list] Delete module version',
    DeleteModuleVersionSuccess = '[module-list] Delete module version failure',
    DeleteModuleVersionFailure = '[module-list] Delete module version failure',

    FetchModuleVersions = '[module-list] Fetch module versions',
    FetchModuleVersionsSuccess = '[module-list] Fetch module versions success',
    FetchModuleVersionsFailure = '[module-list] Fetch module versions failure',

    RestoreState = '[module-list] Restore state',

    ResetImportState = '[module-list] Reset import state',
    ResetModuleErrors = '[module-list] Reset module errors',
    Reset = '[module-list] reset'
}

export const fetchModules = createAction(ActionType.FetchModules);
export const fetchModulesSuccess = createAction(
    ActionType.FetchModulesSuccess,
    props<{ data: PrivateSystemModules }>()
);
export const fetchModulesFailure = createAction(ActionType.FetchModulesFailure);

export const fetchModulesPage = createAction(ActionType.FetchModulesPage, props<{ page?: number }>());
export const fetchModulesPageSuccess = createAction(
    ActionType.FetchModulesPageSuccess,
    props<{ data: PrivateSystemModules; page: number }>()
);
export const fetchModulesPageFailure = createAction(ActionType.FetchModulesPageFailure);

export const fetchModulesTags = createAction(ActionType.FetchModulesTags);
export const fetchModulesTagsSuccess = createAction(ActionType.FetchModulesTagsSuccess, props<{ tags: string[] }>());
export const fetchModulesTagsFailure = createAction(
    ActionType.FetchModulesTagsFailure,
    props<{ error: ErrorResponse }>()
);

export const selectModules = createAction(ActionType.SelectModules, props<{ modules: Module[] }>());
export const selectModulesByIds = createAction(ActionType.SelectModulesByIds, props<{ ids: number[] }>());

export const setGridFiltration = createAction(ActionType.SetFiltrationGrid, props<{ filtration: Filtration }>());
export const setGridFiltrationByTag = createAction(ActionType.SetFiltrationGridByTag, props<{ tag: string }>());
export const setGridSearch = createAction(ActionType.SetSearchGrid, props<{ value: string }>());
export const resetFiltration = createAction(ActionType.ResetFiltration);
export const setGridSorting = createAction(ActionType.SetGridSorting, props<{ sorting: Sorting }>());

export const createModule = createAction(ActionType.CreateModule, props<{ module: ModelsModuleInfo }>());
export const createModuleSuccess = createAction(ActionType.CreateModuleSuccess, props<{ module: ModelsModuleS }>());
export const createModuleFailure = createAction(ActionType.CreateModuleFailure, props<{ error: ErrorResponse }>());

export const importModule = createAction(
    ActionType.ImportModule,
    props<{ name: string; version: string; data: UploadModule }>()
);
export const importModuleSuccess = createAction(ActionType.ImportModuleSuccess);
export const importModuleFailure = createAction(ActionType.ImportModuleFailure, props<{ error: ErrorResponse }>());

export const deleteModule = createAction(ActionType.DeleteModule, props<{ name: string }>());
export const deleteModuleSuccess = createAction(ActionType.DeleteModuleSuccess);
export const deleteModuleFailure = createAction(ActionType.DeleteModuleFailure, props<{ error: ErrorResponse }>());

export const deleteModuleVersion = createAction(
    ActionType.DeleteModuleVersion,
    props<{ name: string; version: string }>()
);
export const deleteModuleVersionSuccess = createAction(ActionType.DeleteModuleVersionSuccess);
export const deleteModuleVersionFailure = createAction(
    ActionType.DeleteModuleVersionFailure,
    props<{ error: ErrorResponse }>()
);

export const fetchModuleVersions = createAction(ActionType.FetchModuleVersions, props<{ name: string }>());
export const fetchModuleVersionsSuccess = createAction(
    ActionType.FetchModuleVersionsSuccess,
    props<{ data: PrivateSystemShortModules }>()
);
export const fetchModuleVersionsFailure = createAction(ActionType.FetchModuleVersionsFailure);

export const restoreState = createAction(ActionType.RestoreState, props<{ restoredState: Partial<State> }>());

export const resetImportState = createAction(ActionType.ResetImportState);
export const resetModuleErrors = createAction(ActionType.ResetModuleErrors);
export const reset = createAction(ActionType.Reset);
