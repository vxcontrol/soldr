import { createReducer, on } from '@ngrx/store';

import {
    DependencyType,
    ErrorResponse,
    ModelsChangelogVersion,
    ModelsDependencyItem,
    ModelsLocale,
    ModelsModuleLocaleDesc,
    ModelsModuleS,
    ModelsModuleSShort,
    PrivatePolicyModulesUpdates
} from '@soldr/api';
import { FilesContent } from '@soldr/models';
import {
    applyChangesToObject,
    clone,
    getChangesArrays,
    getDefaultValueByType,
    getEmptySchema,
    getLocaleDescriptionByValue,
    getNameForNewItem,
    NcFormProperty,
    NcformSchema,
    PropertyType
} from '@soldr/shared';
import { EventConfigurationItemType } from 'libs/features/modules/src/lib/types';

import * as ModuleEditActions from './module-edit.actions';

export const ATOMIC_EVENT = Object.freeze({ actions: [], fields: [] });
export const COMPLEX_EVENT: any = Object.freeze({
    seq: [],
    group_by: [],
    max_count: 0,
    max_time: 0,
    actions: [],
    fields: []
});

function getDefaultEvent(ref: EventConfigurationItemType) {
    switch (true) {
        case /aggregation/.test(ref):
            return { type: 'aggregation', ...COMPLEX_EVENT };
        case /correlation/.test(ref):
            return { type: 'correlation', ...COMPLEX_EVENT };
        case /atomic/.test(ref):
        default:
            return { type: 'atomic', ...ATOMIC_EVENT };
    }
}

function getEventProperties(props: NcFormProperty['properties']): Record<string, any> {
    return Object.keys(props).reduce(
        (acc, k) => ({
            ...acc,
            [k]: props[k].default || getDefaultValueByType(props[k].type)
        }),
        {
            fields: (props.fields || {}).default
        }
    );
}

function getActionProperties(props: NcFormProperty['properties']): Record<string, any> {
    return Object.keys(props).reduce(
        (o, k) => ({
            ...o,
            [k]: props[k].default || getDefaultValueByType(props[k].type)
        }),
        {
            fields: (props.fields || {}).default
        }
    );
}

function refactorIds(obj: Record<string, ModelsModuleLocaleDesc>, arr1: string[], arr2: string[]) {
    const data = clone(obj) as Record<string, ModelsModuleLocaleDesc>;
    const [, , renamed] = getChangesArrays(arr1, arr2);
    for (const oldName of Object.keys(renamed)) {
        const oldValue = data[oldName];
        for (const locale of Object.keys(oldValue as object)) {
            oldValue[locale].title = oldValue[locale].title === oldName ? renamed[oldName] : oldValue[locale].title;
            oldValue[locale].description =
                oldValue[locale].description === oldName ? renamed[oldName] : oldValue[locale].description;
        }
    }

    return data;
}

export const moduleEditFeatureKey = 'module-edit';

export type ValidationState = {
    general: boolean;
    configuration: boolean;
    secureConfiguration: boolean;
    events: boolean;
    actions: boolean;
    fields: boolean;
    dependencies: boolean;
    localization: boolean;
    files: boolean;
    changelog: boolean;
};

export const DEFAULT_ACTION_PRIORITY = 10;

export interface State {
    agentVersions: string[];
    allModules: ModelsModuleS[];
    createDraftError: ErrorResponse;
    deleteError: ErrorResponse;
    deleteVersionError: ErrorResponse;
    exportError: ErrorResponse;
    fetchFilesError: ErrorResponse;
    files: string[];
    filesContent: FilesContent;
    isCreatingDraft: boolean;
    isDeleteNodule: boolean;
    isDeleteNoduleVersion: boolean;
    isExportingModule: boolean;
    isFetchAgentVersions: boolean;
    isFetchAllModules: boolean;
    isFetchFiles: boolean;
    isFetchModuleVersionByName: boolean;
    isLoadingFile: boolean;
    isLoadingModule: boolean;
    isLoadingVersions: boolean;
    isReleasingModule: boolean;
    isSavingFiles: boolean;
    isSavingModule: boolean;
    isUpdatingModuleInPolicies: boolean;
    loadFileErrors: Record<string, ErrorResponse>;
    module: ModelsModuleS;
    moduleVersionByName: Record<string, string[]>;
    original: ModelsModuleS;
    releaseError: ErrorResponse;
    restored: boolean;
    saveError: ErrorResponse;
    saveFilesErrors: Record<string, ErrorResponse>;
    updates: PrivatePolicyModulesUpdates;
    updateInPoliciesError: ErrorResponse;
    validationState: ValidationState;
    versions: ModelsModuleSShort[];
}

export const initialState: State = {
    agentVersions: [],
    allModules: [],
    createDraftError: undefined,
    deleteError: undefined,
    deleteVersionError: undefined,
    exportError: undefined,
    fetchFilesError: undefined,
    files: [],
    filesContent: {},
    isCreatingDraft: false,
    isDeleteNodule: false,
    isDeleteNoduleVersion: false,
    isExportingModule: false,
    isFetchAgentVersions: false,
    isFetchAllModules: false,
    isFetchFiles: false,
    isFetchModuleVersionByName: false,
    isLoadingFile: false,
    isLoadingModule: false,
    isLoadingVersions: false,
    isReleasingModule: false,
    isSavingFiles: false,
    isSavingModule: false,
    isUpdatingModuleInPolicies: false,
    loadFileErrors: {},
    module: undefined,
    moduleVersionByName: {},
    original: undefined,
    releaseError: undefined,
    restored: false,
    saveError: undefined,
    saveFilesErrors: {},
    updates: undefined,
    updateInPoliciesError: undefined,
    validationState: {
        general: true,
        configuration: true,
        secureConfiguration: true,
        events: true,
        actions: true,
        fields: true,
        dependencies: true,
        localization: true,
        files: true,
        changelog: true
    },
    versions: []
};

export const reducer = createReducer(
    initialState,

    // LOADING
    on(ModuleEditActions.fetchModule, (state) => ({ ...state, isLoadingModule: true })),
    on(ModuleEditActions.fetchModuleSuccess, (state, { module, versions, updates, files }) => ({
        ...state,
        isLoadingModule: false,
        module,
        versions,
        updates,
        files,
        original: module
    })),
    on(ModuleEditActions.fetchModuleFailure, (state) => ({ ...state, isLoadingModule: false })),

    on(ModuleEditActions.fetchModuleVersions, (state) => ({ ...state, isLoadingVersions: true })),
    on(ModuleEditActions.fetchModuleVersionsSuccess, (state, { data }) => ({
        ...state,
        isLoadingVersions: false,
        versions: data.modules
    })),
    on(ModuleEditActions.fetchModuleVersionsFailure, (state) => ({ ...state, isLoadingVersions: false })),

    on(ModuleEditActions.fetchModuleUpdatesSuccess, (state, { updates }) => ({ ...state, updates })),

    // OPERATIONS
    on(ModuleEditActions.saveModule, (state) => ({ ...state, isSavingModule: true, error: undefined })),
    on(ModuleEditActions.saveModuleSuccess, (state) => ({ ...state, isSavingModule: false, original: state.module })),
    on(ModuleEditActions.saveModuleFailure, (state, { error }) => ({
        ...state,
        isSavingModule: false,
        saveError: error
    })),

    on(ModuleEditActions.releaseModule, (state) => ({ ...state, isReleasingModule: true, releaseError: undefined })),
    on(ModuleEditActions.releaseModuleSuccess, (state) => ({ ...state, isReleasingModule: false })),
    on(ModuleEditActions.releaseModuleFailure, (state, { error }) => ({
        ...state,
        isReleasingModule: false,
        releaseError: error
    })),

    on(ModuleEditActions.createModuleDraft, (state) => ({
        ...state,
        isCreatingDraft: true,
        createDraftError: undefined
    })),
    on(ModuleEditActions.createModuleDraftSuccess, (state) => ({ ...state, isCreatingDraft: false })),
    on(ModuleEditActions.createModuleDraftFailure, (state, { error }) => ({
        ...state,
        isCreatingDraft: false,
        createDraftError: error
    })),

    on(ModuleEditActions.deleteModule, (state) => ({ ...state, isDeleteNodule: true, deleteError: undefined })),
    on(ModuleEditActions.deleteModuleSuccess, (state) => ({ ...state, isDeleteNodule: false })),
    on(ModuleEditActions.deleteModuleFailure, (state, { error }) => ({
        ...state,
        isDeleteNodule: false,
        deleteError: error
    })),

    on(ModuleEditActions.deleteModuleVersion, (state) => ({
        ...state,
        isDeleteNoduleVersion: true,
        deleteVersionError: undefined
    })),
    on(ModuleEditActions.deleteModuleVersionSuccess, (state) => ({ ...state, isDeleteNoduleVersion: false })),
    on(ModuleEditActions.deleteModuleVersionFailure, (state, { error }) => ({
        ...state,
        isDeleteNoduleVersion: false,
        deleteVersionError: error
    })),

    on(ModuleEditActions.updateModuleInPolicies, (state) => ({
        ...state,
        isUpdatingModuleInPolicies: true,
        updateInPoliciesError: undefined
    })),
    on(ModuleEditActions.updateModuleInPoliciesSuccess, (state) => ({ ...state, isUpdatingModuleInPolicies: false })),
    on(ModuleEditActions.updateModuleInPoliciesFailure, (state, { error }) => ({
        ...state,
        isUpdatingModuleInPolicies: false,
        updateInPoliciesError: error
    })),

    // GENERAL
    on(ModuleEditActions.updateGeneralSection, (state, { info }) => {
        const oldTagsLocale = clone(state.module.locale.tags) as Record<string, ModelsModuleLocaleDesc>;

        const tagsLocale = applyChangesToObject(
            oldTagsLocale,
            Object.keys(state.module.locale.tags),
            info.tags,
            (tag: string) => getLocaleDescriptionByValue(tag)
        );

        return {
            ...state,
            validationState: {
                ...state.validationState,
                general: true
            },
            module: {
                ...state.module,
                info,
                locale: {
                    ...state.module.locale,
                    tags: tagsLocale
                }
            }
        };
    }),

    // CONFIG
    on(ModuleEditActions.addConfigParam, (state) => {
        const newParamName = getNameForNewItem(Object.keys(state.module.config_schema.properties as object), 'param_');

        return {
            ...state,
            module: {
                ...state.module,
                config_schema: {
                    ...state.module.config_schema,
                    properties: {
                        ...state.module.config_schema.properties,
                        [newParamName]: {
                            type: PropertyType.STRING,
                            rules: {},
                            ui: {
                                widgetConfig: {}
                            }
                        } as NcFormProperty
                    }
                },
                default_config: {
                    ...state.module.default_config,
                    [newParamName]: getDefaultValueByType(PropertyType.STRING)
                },
                locale: {
                    ...state.module.locale,
                    config: {
                        ...state.module.locale.config,
                        [newParamName]: getLocaleDescriptionByValue(newParamName)
                    }
                }
            }
        };
    }),
    on(ModuleEditActions.removeConfigParam, (state, { name }) => {
        const props = clone(state.module.config_schema.properties) as Record<string, NcFormProperty>;

        if (name.includes('.')) {
            const parts = name.split('.');
            const nestedObjects = parts.slice(0, -1);
            const propName = parts[parts.length - 1];
            const obj = nestedObjects.reduce((acc, current) => props[current].properties, props);

            delete obj[propName];
        } else {
            delete props[name];
        }

        const locale = clone(state.module.locale.config) as Record<string, ModelsModuleLocaleDesc>;
        delete locale[name];

        const defaultConfig = clone(state.module.default_config) as Record<string, any>;
        if (name.includes('.')) {
            const parts = name.split('.');
            const nestedObjects = parts.slice(0, -1);
            const propName = parts[parts.length - 1];
            const obj = nestedObjects.reduce((acc, current) => acc[current], defaultConfig);

            delete obj[propName];
        } else {
            delete defaultConfig[name];
        }

        return {
            ...state,
            module: {
                ...state.module,
                config_schema: {
                    ...state.module.config_schema,
                    properties: props
                },
                default_config: defaultConfig,
                locale: {
                    ...state.module.locale,
                    config: locale
                }
            }
        };
    }),
    on(ModuleEditActions.removeAllConfigParams, (state) => ({
        ...state,
        module: {
            ...state.module,
            config_schema: getEmptySchema(),
            default_config: {},
            locale: {
                ...state.module.locale,
                config: {}
            }
        }
    })),
    on(ModuleEditActions.updateConfigSchema, (state, { schema }) => {
        const namesOld = Object.keys(state.module.config_schema.properties as object);
        const names = Object.keys(schema.properties as object);
        const processedSchema = clone(schema) as NcformSchema;

        let defaultConfig = clone(state.module.default_config) as Record<string, any>;
        defaultConfig = applyChangesToObject(defaultConfig, namesOld, names, (paramName: string) =>
            getDefaultValueByType(processedSchema.properties[paramName].type)
        );

        for (const paramName of Object.keys(processedSchema.properties)) {
            const type = processedSchema.properties[paramName].type;
            if (state.module.config_schema.properties[paramName]?.type !== type) {
                defaultConfig[paramName] = getDefaultValueByType(type);
            }
        }

        const locale = clone(state.module.locale) as ModelsLocale;
        locale.config = refactorIds(locale.config, namesOld, names);
        locale.config = applyChangesToObject(locale.config, namesOld, names);

        return {
            ...state,
            validationState: {
                ...state.validationState,
                configuration: true
            },
            module: {
                ...state.module,
                config_schema: processedSchema,
                default_config: defaultConfig,
                locale
            }
        };
    }),
    on(ModuleEditActions.updateDefaultConfig, (state, { defaultConfig }) => ({
        ...state,
        validationState: {
            ...state.validationState,
            configuration: true
        },
        module: {
            ...state.module,
            default_config: defaultConfig
        }
    })),

    // SECURE CONFIG
    on(ModuleEditActions.addSecureConfigParam, (state) => {
        const newParamName = getNameForNewItem(
            Object.keys(state.module.secure_config_schema.properties as object),
            'param_'
        );

        return {
            ...state,
            module: {
                ...state.module,
                secure_config_schema: {
                    ...state.module.secure_config_schema,
                    properties: {
                        ...state.module.secure_config_schema.properties,
                        [newParamName]: {
                            type: PropertyType.OBJECT,
                            properties: {
                                value: {
                                    type: PropertyType.STRING,
                                    rules: {},
                                    ui: {
                                        widgetConfig: {}
                                    }
                                },
                                server_only: {
                                    type: PropertyType.BOOLEAN,
                                    value: false,
                                    ui: {
                                        hidden: true
                                    },
                                    enum: [false]
                                }
                            },
                            required: ['server_only', 'value']
                        } as NcFormProperty
                    }
                },
                secure_default_config: {
                    ...state.module.secure_default_config,
                    [newParamName]: {
                        value: getDefaultValueByType(PropertyType.STRING),
                        server_only: false
                    }
                },
                locale: {
                    ...state.module.locale,
                    secure_config: {
                        ...state.module.locale.secure_config,
                        [newParamName]: getLocaleDescriptionByValue(newParamName)
                    }
                }
            }
        };
    }),
    on(ModuleEditActions.removeSecureConfigParam, (state, { name }) => {
        const props = clone(state.module.secure_config_schema.properties) as Record<string, NcFormProperty>;

        if (name.includes('.')) {
            const parts = name.split('.');
            const nestedObjects = parts.slice(0, -1);
            const propName = parts[parts.length - 1];
            const obj = nestedObjects.reduce(
                (acc, current, index) =>
                    index === 0 ? acc[current].properties.value.properties : acc[current].properties,
                props
            );

            delete obj[propName];
        } else {
            delete props[name];
        }

        const locale = clone(state.module.locale.secure_config) as Record<string, ModelsModuleLocaleDesc>;
        delete locale[name];

        const defaultConfig = clone(state.module.secure_default_config) as Record<string, any>;
        if (name.includes('.')) {
            const parts = name.split('.');
            const nestedObjects = parts.slice(0, -1);
            const propName = parts[parts.length - 1];
            const obj = nestedObjects.reduce(
                (acc, current, index) => (index === 0 ? acc[current].value : acc[current]),
                defaultConfig
            );

            delete obj[propName];
        } else {
            delete defaultConfig[name];
        }

        return {
            ...state,
            module: {
                ...state.module,
                secure_config_schema: {
                    ...state.module.secure_config_schema,
                    properties: props
                },
                secure_default_config: defaultConfig,
                locale: {
                    ...state.module.locale,
                    secure_config: locale
                }
            }
        };
    }),
    on(ModuleEditActions.removeAllSecureConfigParams, (state) => ({
        ...state,
        module: {
            ...state.module,
            secure_config_schema: getEmptySchema(),
            secure_default_config: {},
            locale: {
                ...state.module.locale,
                secure_config: {}
            }
        }
    })),
    on(ModuleEditActions.updateSecureConfigSchema, (state, { schema }) => {
        const namesOld = Object.keys(state.module.secure_config_schema.properties as object);
        const names = Object.keys(schema.properties as object);
        const processedSchema = clone(schema) as NcformSchema;

        let defaultConfig = clone(state.module.secure_default_config) as Record<string, any>;
        defaultConfig = applyChangesToObject(defaultConfig, namesOld, names, (paramName: string) =>
            getDefaultValueByType(processedSchema.properties[paramName].properties.value.type)
        );

        for (const paramName of Object.keys(processedSchema.properties)) {
            const type = processedSchema.properties[paramName].properties.value.type;
            defaultConfig[paramName].server_only = processedSchema.properties[paramName].properties.server_only.value;
            if (state.module.secure_config_schema.properties[paramName]?.properties.value.type !== type) {
                defaultConfig[paramName].value = getDefaultValueByType(type);
            }
        }

        const locale = clone(state.module.locale) as ModelsLocale;
        locale.secure_config = refactorIds(locale.secure_config, namesOld, names);
        locale.secure_config = applyChangesToObject(locale.secure_config, namesOld, names);

        return {
            ...state,
            validationState: {
                ...state.validationState,
                secureConfiguration: true
            },
            module: {
                ...state.module,
                secure_config_schema: schema,
                secure_default_config: defaultConfig,
                locale
            }
        };
    }),
    on(ModuleEditActions.updateSecureDefaultConfig, (state, { defaultConfig }) => ({
        ...state,
        validationState: {
            ...state.validationState,
            secureConfiguration: true
        },
        module: {
            ...state.module,
            secure_default_config: defaultConfig
        }
    })),

    // EVENTS
    on(ModuleEditActions.addEvent, (state) => {
        const newEventName = getNameForNewItem(
            Object.keys(state.module.event_config_schema.properties as object),
            'event_'
        );

        return {
            ...state,
            module: {
                ...state.module,
                info: {
                    ...state.module.info,
                    events: [...state.module.info.events, newEventName]
                },
                event_config_schema: {
                    ...state.module.event_config_schema,
                    properties: {
                        ...state.module.event_config_schema.properties,
                        [newEventName]: {
                            type: PropertyType.OBJECT,
                            allOf: [
                                { $ref: '#/definitions/events.atomic' },
                                {
                                    properties: {
                                        fields: {
                                            default: [],
                                            items: {
                                                type: 'string'
                                            },
                                            type: 'array'
                                        }
                                    },
                                    required: ['fields'],
                                    type: PropertyType.OBJECT
                                }
                            ]
                        } as NcFormProperty
                    },
                    required: [...state.module.event_config_schema.required, newEventName]
                },
                default_event_config: {
                    ...state.module.default_event_config,
                    [newEventName]: {
                        actions: [],
                        fields: [],
                        type: 'atomic'
                    }
                },
                locale: {
                    ...state.module.locale,
                    events: {
                        ...state.module.locale.events,
                        [newEventName]: getLocaleDescriptionByValue(newEventName)
                    },
                    event_config: {
                        ...state.module.locale.event_config,
                        [newEventName]: {}
                    }
                }
            }
        };
    }),
    on(ModuleEditActions.removeEvent, (state, { name }) => {
        const props = clone(state.module.event_config_schema.properties) as Record<string, NcFormProperty>;
        delete props[name];

        const locale = clone(state.module.locale.events) as Record<string, ModelsModuleLocaleDesc>;
        delete locale[name];

        const configLocale = clone(state.module.locale.event_config) as ModelsLocale['event_config'];
        delete configLocale[name];

        const defaultConfig = clone(state.module.default_event_config) as Record<string, any>;
        delete defaultConfig[name];

        return {
            ...state,
            module: {
                ...state.module,
                info: {
                    ...state.module.info,
                    events: state.module.info.events.filter((event) => event !== name)
                },
                event_config_schema: {
                    ...state.module.event_config_schema,
                    properties: props,
                    required: state.module.event_config_schema.required.filter((event) => event !== name)
                },
                default_event_config: defaultConfig,
                locale: {
                    ...state.module.locale,
                    events: locale,
                    event_config: configLocale
                }
            }
        };
    }),
    on(ModuleEditActions.removeAllEvents, (state) => ({
        ...state,
        module: {
            ...state.module,
            info: {
                ...state.module.info,
                events: []
            },
            event_config_schema: getEmptySchema(),
            default_event_config: {},
            locale: {
                ...state.module.locale,
                events: {},
                event_config: {}
            }
        }
    })),
    on(ModuleEditActions.updateEventsSchema, (state, { schema }) => {
        const namesOld = Object.keys(state.module.event_config_schema.properties as object);
        const names = Object.keys(schema.properties as object);

        let defaultConfig = clone(state.module.default_event_config) as Record<string, any>;

        defaultConfig = applyChangesToObject(defaultConfig, Object.keys(defaultConfig), names, (eventName: string) =>
            getDefaultEvent(schema.properties[eventName].allOf[0].$ref as EventConfigurationItemType)
        );

        for (const eventName of Object.keys(defaultConfig)) {
            const props = schema.properties[eventName]?.allOf[1].properties || {};
            const eventData = getEventProperties(props);

            const emptyDefaultEventData = {
                ...getDefaultEvent(schema.properties[eventName].allOf[0].$ref as EventConfigurationItemType),
                ...eventData
            } as Record<string, any>;

            defaultConfig[eventName] = applyChangesToObject(
                defaultConfig[eventName] as object,
                Object.keys(defaultConfig[eventName] as object),
                Object.keys(emptyDefaultEventData as object),
                (name) => emptyDefaultEventData[name]
            );

            defaultConfig[eventName].type = emptyDefaultEventData.type;

            if (defaultConfig[eventName].type === 'atomic') {
                defaultConfig[eventName].fields = emptyDefaultEventData.fields;
            }

            for (const prop of Object.keys(defaultConfig[eventName] as object)) {
                if (!defaultConfig[eventName][prop]) {
                    defaultConfig[eventName][prop] = emptyDefaultEventData[prop];
                }
            }
        }

        const locale = clone(state.module.locale) as ModelsLocale;
        locale.events = refactorIds(locale.events, namesOld, names);
        locale.events = applyChangesToObject(locale.events, namesOld, names);
        locale.event_config = applyChangesToObject(locale.event_config, namesOld, names);

        for (const eventName of Object.keys(locale.event_config)) {
            const props = schema.properties[eventName]?.allOf[1].properties || {};
            const eventKeys = Object.keys(props).filter((key) => key !== 'fields');

            locale.event_config[eventName] = refactorIds(
                locale.event_config[eventName],
                Object.keys(locale.event_config[eventName] || {}),
                eventKeys
            );
            locale.event_config[eventName] = applyChangesToObject(
                locale.event_config[eventName],
                Object.keys(locale.event_config[eventName] || {}),
                eventKeys
            );
        }

        return {
            ...state,
            validationState: {
                ...state.validationState,
                events: true
            },
            module: {
                ...state.module,
                info: {
                    ...state.module.info,
                    events: names
                },
                event_config_schema: {
                    ...schema,
                    required: names
                },
                default_event_config: defaultConfig,
                locale
            }
        };
    }),
    on(ModuleEditActions.updateEventsDefaultConfig, (state, { defaultConfig }) => ({
        ...state,
        validationState: {
            ...state.validationState,
            events: true
        },
        module: {
            ...state.module,
            default_event_config: defaultConfig
        }
    })),
    on(ModuleEditActions.addEventKey, (state, { eventName }) => {
        const existedKeys = Object.keys(
            (state.module.event_config_schema.properties[eventName].allOf[1]?.properties || {}) as object
        ).filter((v) => !['fields'].includes(v));
        const newKeyName = getNameForNewItem(existedKeys, 'key_');

        return {
            ...state,
            module: {
                ...state.module,
                event_config_schema: {
                    ...state.module.event_config_schema,
                    properties: {
                        ...state.module.event_config_schema.properties,
                        [eventName]: {
                            allOf: [
                                state.module.event_config_schema.properties[eventName].allOf[0],
                                {
                                    ...state.module.event_config_schema.properties[eventName].allOf[1],
                                    properties: {
                                        ...state.module.event_config_schema.properties[eventName].allOf[1].properties,
                                        [newKeyName]: {
                                            type: PropertyType.STRING,
                                            rules: {},
                                            ui: {
                                                widgetConfig: {}
                                            }
                                        }
                                    }
                                }
                            ]
                        } as NcFormProperty
                    }
                },
                default_event_config: {
                    ...state.module.default_event_config,
                    [eventName]: {
                        ...state.module.default_event_config[eventName],
                        [newKeyName]: ''
                    }
                },
                locale: {
                    ...state.module.locale,
                    event_config: {
                        ...state.module.locale.event_config,
                        [eventName]: {
                            ...state.module.locale.event_config[eventName],
                            [newKeyName]: getLocaleDescriptionByValue(newKeyName)
                        }
                    }
                }
            }
        };
    }),
    on(ModuleEditActions.removeEventKey, (state, { eventName, keyName }) => {
        const props = clone(state.module.event_config_schema.properties) as NcformSchema['properties'];
        const keyParts = keyName.split('.');

        if (keyParts.length > 1) {
            let parent = props[eventName].allOf[1];

            for (const keyPart of keyParts.slice(0, -1)) {
                parent = parent.properties[keyPart];
            }

            delete parent.properties[keyParts[keyParts.length - 1]];
        } else {
            delete props[eventName].allOf[1].properties[keyName];
        }

        const locale = clone(state.module.locale.event_config) as ModelsLocale['event_config'];
        delete locale[eventName][keyName];

        const defaultConfig = clone(state.module.default_event_config) as Record<string, any>;
        delete defaultConfig[eventName][keyName];

        return {
            ...state,
            module: {
                ...state.module,
                event_config_schema: {
                    ...state.module.event_config_schema,
                    properties: props
                },
                default_event_config: defaultConfig,
                locale: {
                    ...state.module.locale,
                    event_config: locale
                }
            }
        };
    }),

    // ACTIONS
    on(ModuleEditActions.addAction, (state) => {
        const newActionName = getNameForNewItem(
            Object.keys(state.module.action_config_schema.properties as object),
            'action_'
        );

        return {
            ...state,
            module: {
                ...state.module,
                info: {
                    ...state.module.info,
                    actions: [...state.module.info.actions, newActionName]
                },
                action_config_schema: {
                    ...state.module.action_config_schema,
                    properties: {
                        ...state.module.action_config_schema.properties,
                        [newActionName]: {
                            allOf: [
                                { $ref: '#/definitions/base.action' },
                                {
                                    properties: {
                                        fields: {
                                            default: [],
                                            items: {
                                                type: 'string'
                                            },
                                            type: 'array'
                                        },
                                        priority: {
                                            default: DEFAULT_ACTION_PRIORITY,
                                            maximum: DEFAULT_ACTION_PRIORITY,
                                            minimum: DEFAULT_ACTION_PRIORITY,
                                            type: 'integer'
                                        }
                                    },
                                    required: ['fields', 'priority'],
                                    type: PropertyType.OBJECT
                                }
                            ]
                        } as NcFormProperty
                    },
                    required: [...state.module.action_config_schema.required, newActionName]
                },
                default_action_config: {
                    ...state.module.default_action_config,
                    [newActionName]: {
                        fields: [],
                        priority: DEFAULT_ACTION_PRIORITY
                    }
                },
                locale: {
                    ...state.module.locale,
                    actions: {
                        ...state.module.locale.actions,
                        [newActionName]: getLocaleDescriptionByValue(newActionName)
                    },
                    action_config: {
                        ...state.module.locale.action_config,
                        [newActionName]: {}
                    }
                }
            }
        };
    }),
    on(ModuleEditActions.removeAction, (state, { name }) => {
        const props = clone(state.module.action_config_schema.properties) as Record<string, NcFormProperty>;
        delete props[name];

        const locale = clone(state.module.locale.actions) as Record<string, ModelsModuleLocaleDesc>;
        delete locale[name];

        const configLocale = clone(state.module.locale.action_config) as ModelsLocale['action_config'];
        delete configLocale[name];

        const defaultConfig = clone(state.module.default_action_config) as Record<string, any>;
        delete defaultConfig[name];

        return {
            ...state,
            module: {
                ...state.module,
                info: {
                    ...state.module.info,
                    actions: state.module.info.actions.filter((action) => action !== name)
                },
                action_config_schema: {
                    ...state.module.action_config_schema,
                    properties: props,
                    required: state.module.action_config_schema.required.filter((event) => event !== name)
                },
                default_action_config: defaultConfig,
                locale: {
                    ...state.module.locale,
                    actions: locale,
                    action_config: configLocale
                }
            }
        };
    }),
    on(ModuleEditActions.removeAllActions, (state) => ({
        ...state,
        module: {
            ...state.module,
            info: {
                ...state.module.info,
                actions: []
            },
            action_config_schema: getEmptySchema(),
            default_action_config: {},
            locale: {
                ...state.module.locale,
                actions: {},
                action_config: {}
            }
        }
    })),
    on(ModuleEditActions.updateActionsSchema, (state, { schema }) => {
        const namesOld = Object.keys(state.module.action_config_schema.properties as object);
        const names = Object.keys(schema.properties as object);

        let defaultConfig = clone(state.module.default_action_config) as Record<string, any>;
        defaultConfig = applyChangesToObject(defaultConfig, namesOld, names);

        for (const actionName of Object.keys(defaultConfig)) {
            const props = schema.properties[actionName]?.allOf[1].properties || {};
            const emptyDefaultActionData: Record<string, any> = {
                fields: [],
                priority: 1,
                ...getActionProperties(props)
            };

            defaultConfig[actionName] = applyChangesToObject(
                defaultConfig[actionName] as object,
                Object.keys(defaultConfig[actionName] as object),
                Object.keys(emptyDefaultActionData as object),
                (name: string) => emptyDefaultActionData[name]
            );

            defaultConfig[actionName].fields = emptyDefaultActionData.fields;
            defaultConfig[actionName].priority = emptyDefaultActionData.priority;

            for (const prop of Object.keys(defaultConfig[actionName] as object)) {
                if (!defaultConfig[actionName][prop]) {
                    defaultConfig[actionName][prop] = emptyDefaultActionData[prop];
                }
            }
        }

        const locale = clone(state.module.locale) as ModelsLocale;
        locale.actions = refactorIds(locale.actions, namesOld, names);
        locale.actions = applyChangesToObject(locale.actions, namesOld, names);
        locale.action_config = applyChangesToObject(locale.action_config, namesOld, names);

        for (const name of names) {
            defaultConfig[name].priority = schema.properties[name].allOf[1].properties.priority.default;
        }

        for (const actionName of Object.keys(locale.action_config)) {
            const props = schema.properties[actionName]?.allOf[1].properties || {};
            const actionKeys = Object.keys(props).filter((key) => !['fields', 'priority'].includes(key));

            locale.action_config[actionName] = refactorIds(
                locale.action_config[actionName],
                Object.keys(locale.action_config[actionName] || {}),
                actionKeys
            );
            locale.action_config[actionName] = applyChangesToObject(
                locale.action_config[actionName],
                Object.keys(locale.action_config[actionName] || {}),
                actionKeys
            );
        }

        return {
            ...state,
            validationState: {
                ...state.validationState,
                actions: true
            },
            module: {
                ...state.module,
                info: {
                    ...state.module.info,
                    actions: names
                },
                action_config_schema: {
                    ...schema,
                    required: names
                },
                default_action_config: defaultConfig,
                locale
            }
        };
    }),
    on(ModuleEditActions.updateActionsDefaultConfig, (state, { defaultConfig }) => ({
        ...state,
        validationState: {
            ...state.validationState,
            actions: true
        },
        module: {
            ...state.module,
            default_action_config: defaultConfig
        }
    })),
    on(ModuleEditActions.addActionKey, (state, { actionName }) => {
        const existedKeys = Object.keys(
            (state.module.action_config_schema.properties[actionName].allOf[1]?.properties || {}) as object
        ).filter((v) => !['priority', 'fields'].includes(v));
        const newKeyName = getNameForNewItem(existedKeys, 'key_');

        return {
            ...state,
            module: {
                ...state.module,
                action_config_schema: {
                    ...state.module.action_config_schema,
                    properties: {
                        ...state.module.action_config_schema.properties,
                        [actionName]: {
                            allOf: [
                                state.module.action_config_schema.properties[actionName].allOf[0],
                                {
                                    ...state.module.action_config_schema.properties[actionName].allOf[1],
                                    properties: {
                                        ...state.module.action_config_schema.properties[actionName].allOf[1].properties,
                                        [newKeyName]: {
                                            type: PropertyType.STRING,
                                            rules: {},
                                            ui: {
                                                widgetConfig: {}
                                            }
                                        }
                                    }
                                }
                            ]
                        } as NcFormProperty
                    }
                },
                default_action_config: {
                    ...state.module.default_action_config,
                    [actionName]: {
                        ...state.module.default_action_config[actionName],
                        [newKeyName]: ''
                    }
                },
                locale: {
                    ...state.module.locale,
                    action_config: {
                        ...state.module.locale.action_config,
                        [actionName]: {
                            ...state.module.locale.action_config[actionName],
                            [newKeyName]: getLocaleDescriptionByValue(newKeyName)
                        }
                    }
                }
            }
        };
    }),
    on(ModuleEditActions.removeActionKey, (state, { actionName, keyName }) => {
        const props = clone(state.module.action_config_schema.properties) as NcformSchema['properties'];
        const keyParts = keyName.split('.');

        if (keyParts.length > 1) {
            let parent = props[actionName].allOf[1];

            for (const keyPart of keyParts.slice(0, -1)) {
                parent = parent.properties[keyPart];
            }

            delete parent.properties[keyParts[keyParts.length - 1]];
        } else {
            delete props[actionName].allOf[1].properties[keyName];
        }

        const locale = clone(state.module.locale.action_config) as ModelsLocale['action_config'];
        delete locale[actionName][keyName];

        const defaultConfig = clone(state.module.default_action_config) as Record<string, any>;
        delete defaultConfig[actionName][keyName];

        return {
            ...state,
            module: {
                ...state.module,
                action_config_schema: {
                    ...state.module.action_config_schema,
                    properties: props
                },
                default_action_config: defaultConfig,
                locale: {
                    ...state.module.locale,
                    action_config: locale
                }
            }
        };
    }),

    // FIELDS
    on(ModuleEditActions.addField, (state) => {
        const newFieldName = getNameForNewItem(Object.keys(state.module.fields_schema.properties as object), 'field_');

        return {
            ...state,
            module: {
                ...state.module,
                info: {
                    ...state.module.info,
                    fields: [...state.module.info.fields, newFieldName]
                },
                fields_schema: {
                    ...state.module.fields_schema,
                    properties: {
                        ...state.module.fields_schema.properties,
                        [newFieldName]: {
                            type: PropertyType.STRING,
                            rules: {},
                            ui: {
                                widgetConfig: {}
                            }
                        } as NcFormProperty
                    }
                },
                locale: {
                    ...state.module.locale,
                    fields: {
                        ...state.module.locale.fields,
                        [newFieldName]: getLocaleDescriptionByValue(newFieldName)
                    }
                }
            }
        };
    }),
    on(ModuleEditActions.removeField, (state, { name }) => {
        const props = clone(state.module.fields_schema.properties) as Record<string, NcFormProperty>;
        delete props[name];

        const locale = clone(state.module.locale.fields) as Record<string, ModelsModuleLocaleDesc>;
        delete locale[name];

        return {
            ...state,
            module: {
                ...state.module,
                info: {
                    ...state.module.info,
                    fields: state.module.info.fields.filter((field) => field !== name)
                },
                fields_schema: {
                    ...state.module.fields_schema,
                    properties: props
                },
                locale: {
                    ...state.module.locale,
                    fields: locale
                }
            }
        };
    }),
    on(ModuleEditActions.removeAllFields, (state) => ({
        ...state,
        module: {
            ...state.module,
            info: {
                ...state.module.info,
                fields: []
            },
            fields_schema: getEmptySchema(),
            locale: {
                ...state.module.locale,
                fields: {}
            }
        }
    })),
    on(ModuleEditActions.updateFieldsSchema, (state, { schema }) => {
        const namesOld = Object.keys(state.module.fields_schema.properties as object);
        const names = Object.keys(schema.properties as object);

        const eventConfigSchema = clone(state.module.event_config_schema) as NcformSchema;
        const actionConfigSchema = clone(state.module.action_config_schema) as NcformSchema;
        const defaultEventConfig = clone(state.module.default_event_config);
        const defaultActionConfig = clone(state.module.default_action_config);

        const renamed = getChangesArrays(namesOld, names)[2];

        for (const eventName of Object.keys(eventConfigSchema.properties || [])) {
            const eventFields = eventConfigSchema.properties[eventName].allOf[1].properties.fields;
            if (!Array.isArray(eventFields?.default)) continue;

            eventFields.default = eventFields.default
                .map((field: string) => renamed[field] || field)
                .filter((field: string) => names.includes(field));
            eventFields.items.enum = eventFields.default;

            if (eventFields.default?.length > 0) {
                eventFields.minItems = eventFields.maxItems = eventFields.default?.length;
            }

            defaultEventConfig[eventName].fields = eventFields.default;
        }

        for (const actionName of Object.keys(actionConfigSchema.properties || [])) {
            const actionFields = actionConfigSchema.properties[actionName].allOf[1].properties.fields;
            if (!Array.isArray(actionFields?.default)) continue;

            actionFields.default = actionFields.default
                .map((field: string) => renamed[field] || field)
                .filter((field: string) => names.includes(field));
            actionFields.items.enum = actionFields.default;

            if (actionFields.default?.length > 0) {
                actionFields.minItems = actionFields.maxItems = actionFields.default?.length;
            }

            defaultActionConfig[actionName].fields = actionFields.default;
        }

        const locale = clone(state.module.locale) as ModelsLocale;
        locale.fields = refactorIds(locale.fields, namesOld, names);
        locale.fields = applyChangesToObject(locale.fields, namesOld, names);

        return {
            ...state,
            validationState: {
                ...state.validationState,
                fields: true
            },
            module: {
                ...state.module,
                info: {
                    ...state.module.info,
                    fields: names
                },
                fields_schema: schema,
                event_config_schema: eventConfigSchema,
                action_config_schema: actionConfigSchema,
                default_event_config: defaultEventConfig,
                default_action_config: defaultActionConfig,
                locale
            }
        };
    }),

    // LOCALIZATION
    on(ModuleEditActions.updateLocalizationModel, (state, { model }) => ({
        ...state,
        validationState: {
            ...state.validationState,
            localization: true
        },
        module: {
            ...state.module,
            locale: {
                ...state.module.locale,
                ...model
            }
        }
    })),

    // FILES
    on(ModuleEditActions.fetchFiles, (state) => ({ ...state, isFetchFiles: true })),
    on(ModuleEditActions.fetchFilesSuccess, (state, { files }) => ({ ...state, isFetchFiles: false, files })),
    on(ModuleEditActions.fetchFilesFailure, (state, { error }) => ({
        ...state,
        isFetchFiles: false,
        fetchFilesError: error
    })),

    on(ModuleEditActions.loadFiles, (state, { paths }) => ({
        ...state,
        isLoadingFile: true,
        filesContent: {
            ...state.filesContent,
            ...paths.reduce(
                (acc, path) => ({
                    ...acc,
                    [path]: {
                        loaded: false,
                        content: ''
                    }
                }),
                {}
            )
        }
    })),
    on(ModuleEditActions.loadFilesSuccess, (state, { files }) => {
        const errors = clone(state.loadFileErrors) as State['loadFileErrors'];

        for (const item of files) {
            delete errors[item.path];
        }

        return {
            ...state,
            isLoadingFile: false,
            loadFileErrors: errors,
            filesContent: {
                ...state.filesContent,
                ...files.reduce(
                    (acc, item) => ({
                        ...acc,
                        [item.path]: {
                            loaded: true,
                            content: item.content
                        }
                    }),
                    {}
                )
            }
        };
    }),
    on(ModuleEditActions.loadFilesFailure, (state, { errors }) => ({
        ...state,
        isLoadingFile: false,
        loadFileErrors: {
            ...state.loadFileErrors,
            ...errors.reduce(
                (acc, item) => ({
                    ...acc,
                    [item.path]: item.error
                }),
                {}
            )
        }
    })),

    on(ModuleEditActions.saveFiles, (state) => ({ ...state, isSavingFiles: true })),
    on(ModuleEditActions.saveFilesSuccess, (state, { files }) => {
        const errors = clone(state.saveFilesErrors) as State['saveFilesErrors'];

        for (const item of files) {
            delete errors[item.path];
        }

        return {
            ...state,
            isSavingFiles: false,
            saveFilesErrors: errors,
            filesContent: {
                ...state.filesContent,
                ...files.reduce(
                    (acc, item) => ({
                        ...acc,
                        [item.path]: {
                            content: item.content,
                            loaded: true
                        }
                    }),
                    {}
                )
            }
        };
    }),
    on(ModuleEditActions.saveFilesFailure, (state, { errors }) => ({
        ...state,
        isSavingFiles: false,
        saveFilesErrors: {
            ...state.saveFilesErrors,
            ...errors.reduce(
                (acc, item) => ({
                    ...acc,
                    [item.path]: item.error
                }),
                {}
            )
        }
    })),

    on(ModuleEditActions.closeFiles, (state, { filePaths }) => {
        const filesContent = clone(state.filesContent) as FilesContent;

        for (const filePath of filePaths) {
            delete filesContent[filePath];
        }

        return {
            ...state,
            filesContent
        };
    }),

    // DEPENDENCIES
    on(ModuleEditActions.changeAgentVersion, (state, { version }) => {
        const staticDependencies = clone(state.module.static_dependencies) as ModelsDependencyItem[];

        return {
            ...state,
            module: {
                ...state.module,
                static_dependencies: [
                    ...staticDependencies.filter((item) => item.type !== DependencyType.AgentVersion),
                    ...(version
                        ? [
                              {
                                  type: DependencyType.AgentVersion,
                                  min_agent_version: version
                              } as ModelsDependencyItem
                          ]
                        : [])
                ]
            }
        };
    }),
    on(ModuleEditActions.addReceiveDataDependency, (state) => {
        const staticDependencies = clone(state.module.static_dependencies) as ModelsDependencyItem[];

        return {
            ...state,
            validationState: {
                ...state.validationState,
                dependencies: true
            },
            module: {
                ...state.module,
                static_dependencies: [
                    ...staticDependencies,
                    {
                        type: DependencyType.ToReceiveData,
                        module_name: '',
                        min_agent_version: ''
                    } as ModelsDependencyItem
                ]
            }
        };
    }),
    on(ModuleEditActions.changeReceiveDataDependencies, (state, { dependencies }) => {
        const staticDependencies = clone(state.module.static_dependencies) as ModelsDependencyItem[];

        return {
            ...state,
            validationState: {
                ...state.validationState,
                dependencies: true
            },
            module: {
                ...state.module,
                static_dependencies: [
                    ...staticDependencies.filter((item) => item.type !== DependencyType.ToReceiveData),
                    ...dependencies
                ]
            }
        };
    }),
    on(ModuleEditActions.addSendDataDependency, (state) => {
        const staticDependencies = clone(state.module.static_dependencies) as ModelsDependencyItem[];

        return {
            ...state,
            module: {
                ...state.module,
                static_dependencies: [
                    ...staticDependencies,
                    {
                        type: DependencyType.ToSendData,
                        module_name: '',
                        min_agent_version: ''
                    } as ModelsDependencyItem
                ]
            }
        };
    }),
    on(ModuleEditActions.changeSendDataDependencies, (state, { dependencies }) => {
        const staticDependencies = clone(state.module.static_dependencies) as ModelsDependencyItem[];

        return {
            ...state,
            validationState: {
                ...state.validationState,
                dependencies: true
            },
            module: {
                ...state.module,
                static_dependencies: [
                    ...staticDependencies.filter((item) => item.type !== DependencyType.ToSendData),
                    ...dependencies
                ]
            }
        };
    }),
    on(ModuleEditActions.fetchAgentVersions, (state) => ({
        ...state,
        isFetchAgentVersions: true
    })),
    on(ModuleEditActions.fetchAgentVersionsSuccess, (state, { agentVersions }) => ({
        ...state,
        isFetchAgentVersions: false,
        agentVersions
    })),
    on(ModuleEditActions.fetchAgentVersionsFailure, (state) => ({
        ...state,
        isFetchAgentVersions: false
    })),
    on(ModuleEditActions.fetchModuleVersionByName, (state) => ({
        ...state,
        isFetchModuleVersionByName: true
    })),
    on(ModuleEditActions.fetchModuleVersionByNameSuccess, (state, { name, versions }) => ({
        ...state,
        isFetchModuleVersionByName: false,
        moduleVersionByName: {
            ...state.moduleVersionByName,
            [name]: versions
        }
    })),
    on(ModuleEditActions.fetchModuleVersionByNameFailure, (state) => ({
        ...state,
        isFetchModuleVersionByName: false
    })),
    on(ModuleEditActions.fetchAllModules, (state) => ({
        ...state,
        isFetchAllModules: true
    })),
    on(ModuleEditActions.fetchAllModulesSuccess, (state, { modules }) => ({
        ...state,
        isFetchAllModules: false,
        allModules: modules
    })),
    on(ModuleEditActions.fetchAllModulesFailure, (state) => ({
        ...state,
        isFetchAllModules: false
    })),

    // CHANGELOG
    on(ModuleEditActions.deleteChangelogRecord, (state, { version }) => {
        const changelog = clone(state.module.changelog) as Record<string, ModelsChangelogVersion>;
        delete changelog[version];

        return {
            ...state,
            module: {
                ...state.module,
                changelog
            }
        };
    }),
    on(ModuleEditActions.updateChangelogSection, (state, { changelog }) => ({
        ...state,
        validationState: {
            ...state.validationState,
            changelog: true
        },
        module: {
            ...state.module,
            changelog
        }
    })),

    // STATE
    on(ModuleEditActions.setValidationState, (state, { section, status }) => ({
        ...state,
        validationState: {
            ...state.validationState,
            [section]: status
        }
    })),
    on(ModuleEditActions.resetOperationErrors, (state) => ({
        ...state,
        releaseError: undefined,
        deleteError: undefined,
        deleteVersionError: undefined
    })),
    on(ModuleEditActions.resetFile, (state) => ({
        ...state,
        isLoadingFile: false,
        isSavingFiles: false,
        filesContent: {},
        loadFileErrors: {},
        saveFilesErrors: {}
    })),
    on(ModuleEditActions.restoreState, (state, { restoredState }) => ({
        ...state,
        restored: true,
        ...restoredState
    })),
    on(ModuleEditActions.reset, () => ({ ...initialState }))
);
