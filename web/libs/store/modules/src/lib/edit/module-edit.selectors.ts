import { createFeatureSelector, createSelector } from '@ngrx/store';

import { DependencyType } from '@soldr/api';
import { compareObjects } from '@soldr/shared';

import * as fromModuleEdit from './module-edit.reducer';

export const selectModuleEditState = createFeatureSelector<fromModuleEdit.State>(fromModuleEdit.moduleEditFeatureKey);

// LOADING
export const selectOriginal = createSelector(selectModuleEditState, (state) => state.original);
export const selectModule = createSelector(selectModuleEditState, (state) => state.module);
export const selectIsLoadingModule = createSelector(selectModuleEditState, (state) => state.isLoadingModule);

export const selectVersions = createSelector(selectModuleEditState, (state) => state.versions);
export const selectIsLoadingModuleVersions = createSelector(selectModuleEditState, (state) => state.isLoadingVersions);

export const selectUpdates = createSelector(selectModuleEditState, (state) => state.updates);

// OPERATIONS
export const selectIsDeletingModule = createSelector(selectModuleEditState, (state) => state.isDeleteNodule);
export const selectIsDeletingModuleVersion = createSelector(
    selectModuleEditState,
    (state) => state.isDeleteNoduleVersion
);
export const selectIsCreatingDraft = createSelector(selectModuleEditState, (state) => state.isCreatingDraft);
export const selectIsReleasingModule = createSelector(selectModuleEditState, (state) => state.isReleasingModule);
export const selectIsSavingModule = createSelector(selectModuleEditState, (state) => state.isSavingModule);
export const selectIsUpdatingModuleInPolicies = createSelector(
    selectModuleEditState,
    (state) => state.isUpdatingModuleInPolicies
);

export const selectCreateDraftError = createSelector(selectModuleEditState, (state) => state.createDraftError);
export const selectDeleteError = createSelector(selectModuleEditState, (state) => state.deleteError);
export const selectDeleteVersionError = createSelector(selectModuleEditState, (state) => state.deleteVersionError);
export const selectReleaseError = createSelector(selectModuleEditState, (state) => state.releaseError);
export const selectSaveError = createSelector(selectModuleEditState, (state) => state.saveError);
export const selectUpdateModuleInPoliciesError = createSelector(
    selectModuleEditState,
    (state) => state.updateInPoliciesError
);

// GENERAL
export const selectIsDirtyGeneral = createSelector(
    selectModule,
    selectOriginal,
    (module, original) => module?.info && original?.info && !compareObjects(module?.info, original?.info)
);

// CONFIG
export const selectConfigSection = createSelector(selectModule, (module) => module.config_schema);
export const selectIsDirtyConfigSchema = createSelector(
    selectModule,
    selectOriginal,
    (module, original) =>
        module?.config_schema &&
        original?.config_schema &&
        !compareObjects(module?.config_schema, original?.config_schema)
);
export const selectIsDirtyDefaultConfig = createSelector(
    selectModule,
    selectOriginal,
    (module, original) =>
        module?.default_config &&
        original?.default_config &&
        !compareObjects(module?.default_config, original?.default_config)
);
export const selectIsDirtyConfig = createSelector(
    selectIsDirtyConfigSchema,
    selectIsDirtyDefaultConfig,
    (isDirtyConfigSchema, isDirtyDefaultConfig) => isDirtyConfigSchema || isDirtyDefaultConfig
);

// SECURE CONFIG
export const selectSecureConfigSection = createSelector(selectModule, (module) => module.secure_config_schema);
export const selectIsDirtySecureConfigSchema = createSelector(
    selectModule,
    selectOriginal,
    (module, original) =>
        module?.secure_config_schema &&
        original?.secure_config_schema &&
        !compareObjects(module?.secure_config_schema, original?.secure_config_schema)
);
export const selectIsDirtySecureDefaultConfig = createSelector(
    selectModule,
    selectOriginal,
    (module, original) =>
        module?.secure_default_config &&
        original?.secure_default_config &&
        !compareObjects(module?.secure_default_config, original?.secure_default_config)
);
export const selectIsDirtySecureConfig = createSelector(
    selectIsDirtySecureConfigSchema,
    selectIsDirtySecureDefaultConfig,
    (isDirtySecureConfigSchema, isDirtySecureDefaultConfig) => isDirtySecureConfigSchema || isDirtySecureDefaultConfig
);
export const selectChangedSecureParams = createSelector(
    selectSecureConfigSection,
    selectOriginal,
    (schema, originalModule) =>
        Object.keys(schema.properties).filter(
            (paramName) =>
                !originalModule.secure_config_schema.properties[paramName] ||
                !compareObjects(schema.properties[paramName], originalModule.secure_config_schema.properties[paramName])
        )
);

// EVENTS
export const selectEventsConfigSection = createSelector(selectModule, (module) => module.event_config_schema);
export const selectIsDirtyEventsConfigSchema = createSelector(
    selectModule,
    selectOriginal,
    (module, original) =>
        module?.event_config_schema &&
        original?.event_config_schema &&
        !compareObjects(module?.event_config_schema, original?.event_config_schema)
);
export const selectIsDirtyDefaultEventsConfig = createSelector(
    selectModule,
    selectOriginal,
    (module, original) =>
        module?.default_event_config &&
        original?.default_event_config &&
        !compareObjects(module?.default_event_config, original?.default_event_config)
);
export const selectIsDirtyEventsConfig = createSelector(
    selectIsDirtyEventsConfigSchema,
    selectIsDirtyDefaultEventsConfig,
    (isDirtyEventsConfigSchema, isDirtyDefaultEventsConfig) => isDirtyEventsConfigSchema || isDirtyDefaultEventsConfig
);
export const selectChangedEvents = createSelector(selectEventsConfigSection, selectOriginal, (schema, originalModule) =>
    Object.keys(schema.properties).filter(
        (eventName) =>
            !originalModule.event_config_schema.properties[eventName] ||
            !compareObjects(schema.properties[eventName], originalModule.event_config_schema.properties[eventName])
    )
);

// ACTIONS
export const selectActionsConfigSection = createSelector(selectModule, (module) => module.action_config_schema);
export const selectIsDirtyActionsConfigSchema = createSelector(
    selectModule,
    selectOriginal,
    (module, original) =>
        module?.action_config_schema &&
        original?.action_config_schema &&
        !compareObjects(module?.action_config_schema, original?.action_config_schema)
);
export const selectIsDirtyDefaultActionsConfig = createSelector(
    selectModule,
    selectOriginal,
    (module, original) =>
        module?.default_action_config &&
        original?.default_action_config &&
        !compareObjects(module?.default_action_config, original?.default_action_config)
);
export const selectIsDirtyActionsConfig = createSelector(
    selectIsDirtyActionsConfigSchema,
    selectIsDirtyDefaultActionsConfig,
    (isDirtyActionsConfigSchema, isDirtyDefaultActionsConfig) =>
        isDirtyActionsConfigSchema || isDirtyDefaultActionsConfig
);
export const selectChangedActions = createSelector(
    selectActionsConfigSection,
    selectOriginal,
    (schema, originalModule) =>
        Object.keys(schema.properties).filter(
            (actionName) =>
                !originalModule.action_config_schema.properties[actionName] ||
                !compareObjects(
                    schema.properties[actionName],
                    originalModule.action_config_schema.properties[actionName]
                )
        )
);

// FIELDS
export const selectFieldsConfigSection = createSelector(selectModule, (module) => module.fields_schema);
export const selectIsDirtyFieldsConfigSchema = createSelector(
    selectModule,
    selectOriginal,
    (module, original) =>
        module?.fields_schema &&
        original?.fields_schema &&
        !compareObjects(module?.fields_schema, original?.fields_schema)
);
export const selectUnusedFields = createSelector(selectModule, (module) => {
    const usedFields = [
        ...Object.keys(module?.default_action_config || {}).reduce(
            (acc, actionName) => [...new Set([...acc, ...module?.default_action_config[actionName].fields])],
            [] as string[]
        ),
        ...Object.keys(module?.default_event_config || {}).reduce(
            (acc, eventName) => [...new Set([...acc, ...module?.default_event_config[eventName].fields])],
            [] as string[]
        )
    ];

    return module?.info.fields.filter((field) => !usedFields.includes(field)) || [];
});
export const selectUnusedRequiredFields = createSelector(selectModule, (module) => {
    const events = Object.values(module.default_event_config).filter((event) => event.type === 'atomic');
    const actions = Object.values(module.default_action_config);
    const requiredFields = module.fields_schema?.required || [];

    return requiredFields.filter((field: string) =>
        [...events, ...actions].some(
            (item) => item.fields && Array.isArray(item.fields) && !item.fields.includes(field)
        )
    );
});

// LOCALIZATION
export const selectLocalizationSection = createSelector(selectModule, (module) => module.locale);
export const selectIsDirtyLocalizationModel = createSelector(
    selectModule,
    selectOriginal,
    (module, original) => module?.locale && original?.locale && !compareObjects(module?.locale, original?.locale)
);

// FILES
export const selectFiles = createSelector(selectModuleEditState, (state) => state.files);
export const selectIsFetchFiles = createSelector(selectModuleEditState, (state) => state.isFetchFiles);
export const selectFetchFilesError = createSelector(selectModuleEditState, (state) => state.fetchFilesError);
export const selectIsLoadingFile = createSelector(selectModuleEditState, (state) => state.isLoadingFile);
export const selectOpenedFiles = createSelector(selectModuleEditState, (state) => state.filesContent);
export const selectIsSavingFiles = createSelector(selectModuleEditState, (state) => state.isSavingFiles);
export const selectSaveFileErrors = createSelector(selectModuleEditState, (state) => state.saveFilesErrors);
export const selectLoadFileErrors = createSelector(selectModuleEditState, (state) => state.loadFileErrors);

// DEPENDENCIES
export const selectDependencies = createSelector(selectModule, (module) => module.static_dependencies);
export const selectAgentVersionDependency = createSelector(selectDependencies, (dependencies) =>
    dependencies.find(({ type }) => type === DependencyType.AgentVersion)
);
export const selectReceiveDataDependencies = createSelector(selectDependencies, (dependencies) =>
    dependencies.filter(({ type }) => type === DependencyType.ToReceiveData)
);
export const selectSendDataDependencies = createSelector(selectDependencies, (dependencies) =>
    dependencies.filter(({ type }) => type === DependencyType.ToSendData)
);
export const selectAllModules = createSelector(selectModuleEditState, (state) => state.allModules);
export const selectIsFetchAllModules = createSelector(selectModuleEditState, (state) => state.isFetchAllModules);
export const selectAgentVersions = createSelector(selectModuleEditState, (state) => state.agentVersions);
export const selectIsFetchAgentVersions = createSelector(selectModuleEditState, (state) => state.isFetchAgentVersions);
export const selectModuleVersionsByName = createSelector(selectModuleEditState, (state) => state.moduleVersionByName);
export const selectIsFetchModuleVersionsByName = createSelector(
    selectModuleEditState,
    (state) => state.isFetchModuleVersionByName
);
export const selectIsDirtyStaticDependenciesModel = createSelector(
    selectModule,
    selectOriginal,
    (module, original) =>
        module?.static_dependencies &&
        original?.static_dependencies &&
        !compareObjects(module?.static_dependencies, original?.static_dependencies)
);

// CHANGELOG
export const selectChangelog = createSelector(selectModule, (module) => module.changelog);
export const selectIsDirtyChangelogModel = createSelector(
    selectModule,
    selectOriginal,
    (module, original) =>
        module?.changelog && original?.changelog && !compareObjects(module?.changelog, original?.changelog)
);

// STATE
export const selectValidationState = createSelector(selectModuleEditState, (state) => state.validationState);
export const selectIsValidGeneral = createSelector(selectValidationState, (validationState) => validationState.general);
export const selectIsValidConfiguration = createSelector(
    selectValidationState,
    (validationState) => validationState.configuration
);
export const selectIsValidSecureConfiguration = createSelector(
    selectValidationState,
    (validationState) => validationState.secureConfiguration
);
export const selectIsValidEvents = createSelector(selectValidationState, (validationState) => validationState.events);
export const selectIsValidActions = createSelector(selectValidationState, (validationState) => validationState.actions);
export const selectIsValidFields = createSelector(selectValidationState, (validationState) => validationState.fields);
export const selectIsValidDependencies = createSelector(
    selectValidationState,
    (validationState) => validationState.dependencies
);
export const selectIsValidLocalization = createSelector(
    selectValidationState,
    (validationState) => validationState.localization
);
export const selectIsValidFiles = createSelector(selectValidationState, (validationState) => validationState.files);
export const selectIsValidChangelog = createSelector(
    selectValidationState,
    (validationState) => validationState.changelog
);

export const selectIsDirty = createSelector(
    selectIsDirtyGeneral,
    selectIsDirtyConfig,
    selectIsDirtySecureConfig,
    selectIsDirtyEventsConfig,
    selectIsDirtyActionsConfig,
    selectIsDirtyFieldsConfigSchema,
    selectIsDirtyLocalizationModel,
    selectIsDirtyStaticDependenciesModel,
    selectIsDirtyChangelogModel,
    (
        isDirtyGeneral,
        isDirtyConfig,
        isDirtySecureConfig,
        isDirtyEventsConfig,
        isDirtyActionsConfig,
        isDirtyFieldsConfigSchema,
        isDirtyLocalizationModel,
        isDirtyStaticDependenciesModel,
        isDirtyChangelogModel
    ) =>
        isDirtyGeneral ||
        isDirtyConfig ||
        isDirtySecureConfig ||
        isDirtyEventsConfig ||
        isDirtyActionsConfig ||
        isDirtyFieldsConfigSchema ||
        isDirtyLocalizationModel ||
        isDirtyStaticDependenciesModel ||
        isDirtyChangelogModel
);

export const selectRestored = createSelector(selectModuleEditState, (state) => state.restored);
