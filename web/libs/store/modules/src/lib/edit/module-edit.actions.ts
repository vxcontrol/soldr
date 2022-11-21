import { createAction, props } from '@ngrx/store';

import {
    ErrorResponse,
    ModelsChangelog,
    ModelsChangelogVersion,
    ModelsDependencyItem,
    ModelsLocale,
    ModelsModuleInfo,
    ModelsModuleS,
    ModelsModuleSShort,
    PrivatePolicyModulesUpdates,
    PrivateSystemShortModules
} from '@soldr/api';
import { NcformSchema } from '@soldr/shared';

import { State } from '../list/module-list.reducer';

import { ValidationState } from './module-edit.reducer';

export enum ActionType {
    // LOADING
    FetchModule = '[module-edit] Fetch module',
    FetchModuleSuccess = '[module-edit] Fetch module - Success',
    FetchModuleFailure = '[module-edit] Fetch module - Failure',

    FetchModuleVersions = '[module-edit] Fetch module versions',
    FetchModuleVersionsSuccess = '[module-edit] Fetch module versions - Success',
    FetchModuleVersionsFailure = '[module-edit] Fetch module versions - Failure',

    FetchModuleUpdates = '[module-edit] Fetch module updates',
    FetchModuleUpdatesSuccess = '[module-edit] Fetch module updates - Success',
    FetchModuleUpdatesFailure = '[module-edit] Fetch module updates - Failure',

    // OPERATIONS
    SaveModule = '[module-edit] Save module',
    SaveModuleSuccess = '[module-edit] Save module - Success',
    SaveModuleFailure = '[module-edit] Save module - Failure',

    ReleaseModule = '[module-edit] Release module',
    ReleaseModuleSuccess = '[module-edit] Release module - Success',
    ReleaseModuleFailure = '[module-edit] Release module - Failure',

    CreateModuleDraft = '[module-edit] Create module draft',
    CreateModuleDraftSuccess = '[module-edit] Create module draft - Success',
    CreateModuleDraftFailure = '[module-edit] Create module draft - Failure',

    UpdateModuleInPolicies = '[module-edit] Update module in policies',
    UpdateModuleInPoliciesSuccess = '[module-edit] Update module in policies - Success',
    UpdateModuleInPoliciesFailure = '[module-edit] Update module in policies - Failure',

    DeleteModule = '[module-edit] Delete module',
    DeleteModuleSuccess = '[module-edit] Delete module success',
    DeleteModuleFailure = '[module-edit] Delete module failure',

    DeleteModuleVersion = '[module-edit] Delete module version',
    DeleteModuleVersionSuccess = '[module-edit] Delete module version failure',
    DeleteModuleVersionFailure = '[module-edit] Delete module version failure',

    // GENERAL
    UpdateGeneralSection = '[module-edit] Update general section',

    // CONFIG
    AddConfigParam = '[module-edit] Add config param',
    RemoveConfigParam = '[module-edit] Remove config param',
    RemoveAllConfigParams = '[module-edit] Remove all config params',
    UpdateConfigSchema = '[module-edit] Update config schema',
    UpdateDefaultConfig = '[module-edit] Update default config',

    // SECURE CONFIG
    AddSecureConfigParam = '[module-edit] Add secure config param',
    RemoveSecureConfigParam = '[module-edit] Remove secure config param',
    RemoveAllSecureConfigParams = '[module-edit] Remove all secure config params',
    UpdateSecureConfigSchema = '[module-edit] Update secure config schema',
    UpdateSecureDefaultConfig = '[module-edit] Update secure default config',

    // EVENTS
    AddEvent = '[module-edit] Add event',
    RemoveEvent = '[module-edit] Remove event',
    RemoveAllEvents = '[module-edit] Remove all events',
    UpdateEventsSchema = '[module-edit] Update events schema',
    UpdateEventsDefaultConfig = '[module-edit] Update events default config',
    AddEventKey = '[module-edit] Add event key',
    RemoveEventKey = '[module-edit] Remove event key',

    // ACTIONS
    AddAction = '[module-edit] Add action',
    RemoveAction = '[module-edit] Remove action',
    RemoveAllActions = '[module-edit] Remove all actions',
    UpdateActionsSchema = '[module-edit] Update actions schema',
    UpdateActionsDefaultConfig = '[module-edit] Update actions default config',
    AddActionKey = '[module-edit] Add action key',
    RemoveActionKey = '[module-edit] Remove action key',

    // FIELDS
    AddField = '[module-edit] Add field',
    RemoveField = '[module-edit] Remove field',
    RemoveAllFields = '[module-edit] Remove all fields',
    UpdateFieldsSchema = '[module-edit] Update fields schema',

    // DEPENDENCIES
    ChangeAgentVersionDependency = '[module-edit] Change agent version dependency',
    AddReceiveDataDependency = '[module-edit] Add receive data dependency',
    ChangeReceiveDataDependencies = '[module-edit] Change receive data dependencies',
    AddSendDataDependency = '[module-edit] Add send data dependency',
    ChangeSendDataDependencies = '[module-edit] Change send data dependencies',

    FetchAllModules = '[module-edit] Fetch all modules',
    FetchAllModulesSuccess = '[module-edit] Fetch all modules - Success',
    FetchAllModulesFailure = '[module-edit] Fetch all modules - Failure',

    FetchAgentVersions = '[module-edit] Fetch agent versions',
    FetchAgentVersionsSuccess = '[module-edit] Fetch agent versions - Success',
    FetchAgentVersionsFailure = '[module-edit] Fetch agent versions - Failure',

    FetchModuleVersionsByName = '[module-edit] Fetch module versions by name',
    FetchModuleVersionsByNameSuccess = '[module-edit] Fetch module versions by name - Success',
    FetchModuleVersionsByNameFailure = '[module-edit] Fetch module versions by name - Failure',

    // LOCALIZATION
    UpdateLocalizationModel = '[module-edit] Update localization model',

    // FILES
    FetchFiles = '[module-edit] Fetch files',
    FetchFilesSuccess = '[module-edit] Fetch files - Success',
    FetchFilesFailure = '[module-edit] Fetch files - Failure',

    LoadFiles = '[module-edit] Load files',
    LoadFilesSuccess = '[module-edit] Load files - Success',
    LoadFilesFailure = '[module-edit] Load files - Failure',

    SaveFiles = '[module-edit] Save files',
    SaveFilesSuccess = '[module-edit] Save files - Success',
    SaveFilesFailure = '[module-edit] Save files - Failure',

    CloseFiles = '[module-edit] Close files',

    // CHANGELOG
    DeleteChangelogRecord = '[module-edit] Delete changelog record',
    UpdateChangelogSection = '[module-edit] Update changelog section',

    // STATE
    SetValidationState = '[module-edit] Set validation state',
    ResetOperationErrors = '[module-edit] Reset operation errors',
    ResetFile = '[module-edit] Reset file',
    RestoreState = '[module-edit] Restore state',
    Reset = '[module-edit] reset'
}

// LOADING
export const fetchModule = createAction(ActionType.FetchModule, props<{ name: string; version: string }>());
export const fetchModuleSuccess = createAction(
    ActionType.FetchModuleSuccess,
    props<{
        module: ModelsModuleS;
        versions: ModelsModuleSShort[];
        updates: PrivatePolicyModulesUpdates;
        files: string[];
    }>()
);
export const fetchModuleFailure = createAction(ActionType.FetchModuleFailure);

export const fetchModuleVersions = createAction(ActionType.FetchModuleVersions, props<{ name: string }>());
export const fetchModuleVersionsSuccess = createAction(
    ActionType.FetchModuleVersionsSuccess,
    props<{ data: PrivateSystemShortModules }>()
);
export const fetchModuleVersionsFailure = createAction(ActionType.FetchModuleVersionsFailure);

export const fetchModuleUpdates = createAction(
    ActionType.FetchModuleUpdates,
    props<{ name: string; version: string }>()
);
export const fetchModuleUpdatesSuccess = createAction(
    ActionType.FetchModuleUpdatesSuccess,
    props<{ updates: PrivatePolicyModulesUpdates }>()
);
export const fetchModuleUpdatesFailure = createAction(ActionType.FetchModuleUpdatesFailure);

// OPERATIONS
export const releaseModule = createAction(
    ActionType.ReleaseModule,
    props<{ name: string; version: string; module: ModelsModuleS }>()
);
export const releaseModuleSuccess = createAction(ActionType.ReleaseModuleSuccess);
export const releaseModuleFailure = createAction(ActionType.ReleaseModuleFailure, props<{ error: ErrorResponse }>());

export const createModuleDraft = createAction(
    ActionType.CreateModuleDraft,
    props<{ name: string; version: string; changelog: ModelsChangelogVersion }>()
);
export const createModuleDraftSuccess = createAction(ActionType.CreateModuleDraftSuccess);
export const createModuleDraftFailure = createAction(
    ActionType.CreateModuleDraftFailure,
    props<{ error: ErrorResponse }>()
);

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

export const updateModuleInPolicies = createAction(
    ActionType.UpdateModuleInPolicies,
    props<{ name: string; version: string }>()
);
export const updateModuleInPoliciesSuccess = createAction(ActionType.UpdateModuleInPoliciesSuccess);
export const updateModuleInPoliciesFailure = createAction(
    ActionType.UpdateModuleInPoliciesFailure,
    props<{ error: ErrorResponse }>()
);

export const saveModule = createAction(ActionType.SaveModule);
export const saveModuleSuccess = createAction(ActionType.SaveModuleSuccess);
export const saveModuleFailure = createAction(ActionType.SaveModuleFailure, props<{ error: ErrorResponse }>());

// GENERAL
export const updateGeneralSection = createAction(ActionType.UpdateGeneralSection, props<{ info: ModelsModuleInfo }>());

// CONFIG
export const addConfigParam = createAction(ActionType.AddConfigParam);
export const removeConfigParam = createAction(ActionType.RemoveConfigParam, props<{ name: string }>());
export const removeAllConfigParams = createAction(ActionType.RemoveAllConfigParams);
export const updateConfigSchema = createAction(ActionType.UpdateConfigSchema, props<{ schema: NcformSchema }>());
export const updateDefaultConfig = createAction(
    ActionType.UpdateDefaultConfig,
    props<{ defaultConfig: Record<string, any> }>()
);

// SECURE CONFIG
export const addSecureConfigParam = createAction(ActionType.AddSecureConfigParam);
export const removeSecureConfigParam = createAction(ActionType.RemoveSecureConfigParam, props<{ name: string }>());
export const removeAllSecureConfigParams = createAction(ActionType.RemoveAllSecureConfigParams);
export const updateSecureConfigSchema = createAction(
    ActionType.UpdateSecureConfigSchema,
    props<{ schema: NcformSchema }>()
);
export const updateSecureDefaultConfig = createAction(
    ActionType.UpdateSecureDefaultConfig,
    props<{ defaultConfig: Record<string, any> }>()
);

// EVENTS
export const addEvent = createAction(ActionType.AddEvent);
export const removeEvent = createAction(ActionType.RemoveEvent, props<{ name: string }>());
export const removeAllEvents = createAction(ActionType.RemoveAllEvents);
export const updateEventsSchema = createAction(ActionType.UpdateEventsSchema, props<{ schema: NcformSchema }>());
export const updateEventsDefaultConfig = createAction(
    ActionType.UpdateEventsDefaultConfig,
    props<{ defaultConfig: Record<string, any> }>()
);
export const addEventKey = createAction(ActionType.AddEventKey, props<{ eventName: string }>());
export const removeEventKey = createAction(ActionType.RemoveEventKey, props<{ eventName: string; keyName: string }>());

// ACTIONS
export const addAction = createAction(ActionType.AddAction);
export const removeAction = createAction(ActionType.RemoveAction, props<{ name: string }>());
export const removeAllActions = createAction(ActionType.RemoveAllActions);
export const updateActionsSchema = createAction(ActionType.UpdateActionsSchema, props<{ schema: NcformSchema }>());
export const updateActionsDefaultConfig = createAction(
    ActionType.UpdateActionsDefaultConfig,
    props<{ defaultConfig: Record<string, any> }>()
);
export const addActionKey = createAction(ActionType.AddActionKey, props<{ actionName: string }>());
export const removeActionKey = createAction(
    ActionType.RemoveActionKey,
    props<{ actionName: string; keyName: string }>()
);

// FIELDS
export const addField = createAction(ActionType.AddField);
export const removeField = createAction(ActionType.RemoveField, props<{ name: string }>());
export const removeAllFields = createAction(ActionType.RemoveAllFields);
export const updateFieldsSchema = createAction(ActionType.UpdateFieldsSchema, props<{ schema: NcformSchema }>());

// LOCALIZATION
export const updateLocalizationModel = createAction(
    ActionType.UpdateLocalizationModel,
    props<{ model: Partial<ModelsLocale> }>()
);

// FILES
export const fetchFiles = createAction(ActionType.FetchFiles, props<{ name: string; version: string }>());
export const fetchFilesSuccess = createAction(ActionType.FetchFilesSuccess, props<{ files: string[] }>());
export const fetchFilesFailure = createAction(ActionType.FetchFilesFailure, props<{ error: ErrorResponse }>());

export const loadFiles = createAction(
    ActionType.LoadFiles,
    props<{ moduleName: string; version: string; paths: string[] }>()
);
export const loadFilesSuccess = createAction(
    ActionType.LoadFilesSuccess,
    props<{ files: { path: string; content: string }[] }>()
);
export const loadFilesFailure = createAction(
    ActionType.LoadFilesFailure,
    props<{ errors: { path: string; error: ErrorResponse }[] }>()
);

export const saveFiles = createAction(
    ActionType.SaveFiles,
    props<{ moduleName: string; version: string; files: { path: string; content: string }[] }>()
);
export const saveFilesSuccess = createAction(
    ActionType.SaveFilesSuccess,
    props<{ files: { path: string; content: string }[] }>()
);
export const saveFilesFailure = createAction(
    ActionType.SaveFilesFailure,
    props<{ errors: { path: string; error: ErrorResponse }[] }>()
);

export const closeFiles = createAction(ActionType.CloseFiles, props<{ filePaths: string[] }>());

// DEPENDENCIES
export const changeAgentVersion = createAction(ActionType.ChangeAgentVersionDependency, props<{ version: string }>());
export const addReceiveDataDependency = createAction(ActionType.AddReceiveDataDependency);
export const changeReceiveDataDependencies = createAction(
    ActionType.ChangeReceiveDataDependencies,
    props<{ dependencies: ModelsDependencyItem[] }>()
);
export const addSendDataDependency = createAction(ActionType.AddSendDataDependency);
export const changeSendDataDependencies = createAction(
    ActionType.ChangeSendDataDependencies,
    props<{ dependencies: ModelsDependencyItem[] }>()
);

export const fetchAllModules = createAction(ActionType.FetchAllModules);
export const fetchAllModulesSuccess = createAction(
    ActionType.FetchAllModulesSuccess,
    props<{ modules: ModelsModuleS[] }>()
);
export const fetchAllModulesFailure = createAction(
    ActionType.FetchAllModulesFailure,
    props<{ error: ErrorResponse }>()
);

export const fetchAgentVersions = createAction(ActionType.FetchAgentVersions);
export const fetchAgentVersionsSuccess = createAction(
    ActionType.FetchAgentVersionsSuccess,
    props<{ agentVersions: string[] }>()
);
export const fetchAgentVersionsFailure = createAction(
    ActionType.FetchAgentVersionsFailure,
    props<{ error: ErrorResponse }>()
);

export const fetchModuleVersionByName = createAction(ActionType.FetchModuleVersionsByName, props<{ name: string }>());
export const fetchModuleVersionByNameSuccess = createAction(
    ActionType.FetchModuleVersionsByNameSuccess,
    props<{ name: string; versions: string[] }>()
);
export const fetchModuleVersionByNameFailure = createAction(
    ActionType.FetchModuleVersionsByNameFailure,
    props<{ error: ErrorResponse }>()
);

// CHANGELOG
export const setValidationState = createAction(
    ActionType.SetValidationState,
    props<{ section: keyof ValidationState; status: boolean }>()
);
export const deleteChangelogRecord = createAction(ActionType.DeleteChangelogRecord, props<{ version: string }>());
export const updateChangelogSection = createAction(
    ActionType.UpdateChangelogSection,
    props<{ changelog: ModelsChangelog }>()
);

// STATE
export const resetOperationErrors = createAction(ActionType.ResetOperationErrors);
export const resetFile = createAction(ActionType.ResetFile);
export const restoreState = createAction(ActionType.RestoreState, props<{ restoredState: Partial<State> }>());
export const reset = createAction(ActionType.Reset);
