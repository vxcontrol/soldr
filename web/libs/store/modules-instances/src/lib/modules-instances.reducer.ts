import { createReducer, on } from '@ngrx/store';

import { ErrorResponse, ModelsModuleSShort } from '@soldr/api';
import { Event, privateEventsToModels } from '@soldr/models';
import { Filtration, Sorting, ViewMode } from '@soldr/shared';

import * as Actions from './modules-instances.actions';

export const modulesInstancesFeatureKey = 'modulesInstances';

export interface State {
    changeVersionError: ErrorResponse;
    deleteModuleError: ErrorResponse;
    disableModuleError: ErrorResponse;
    enableModuleError: ErrorResponse;
    entityId: number;
    events: Event[];
    eventsGridFiltration: Filtration[];
    eventsGridSearch: string;
    eventsPage: number;
    eventsSorting: Sorting | Record<never, any>;
    isChangingVersion: boolean;
    isDeletingModule: boolean;
    isDisablingModule: boolean;
    isEnablingModule: boolean;
    isLoadingEvents: boolean;
    isLoadingModuleEventsFilterItems: boolean;
    isLoadingModuleVersions: boolean;
    isSavingModuleConfig: boolean;
    isUpdatingModule: boolean;
    eventFilterItemAgentNames: string[];
    eventFilterItemGroupIds: string[];
    moduleName: string;
    moduleVersions: ModelsModuleSShort[];
    savingModuleError: ErrorResponse;
    totalEvents: number;
    updateModuleError: ErrorResponse;
    viewMode: ViewMode;
}

export const initialState: State = {
    changeVersionError: undefined,
    deleteModuleError: undefined,
    disableModuleError: undefined,
    enableModuleError: undefined,
    entityId: undefined,
    events: [],
    eventsGridFiltration: [],
    eventsGridSearch: '',
    eventsPage: 0,
    eventsSorting: {},
    isChangingVersion: false,
    isDeletingModule: false,
    isDisablingModule: false,
    isEnablingModule: false,
    isLoadingEvents: false,
    isLoadingModuleEventsFilterItems: false,
    isLoadingModuleVersions: false,
    isSavingModuleConfig: false,
    isUpdatingModule: false,
    eventFilterItemAgentNames: [],
    eventFilterItemGroupIds: [],
    moduleName: undefined,
    moduleVersions: [],
    savingModuleError: undefined,
    totalEvents: 0,
    updateModuleError: undefined,
    viewMode: undefined
};

export const reducer = createReducer(
    initialState,

    on(Actions.init, (state, { viewMode, entityId, moduleName }) => ({ ...state, viewMode, entityId, moduleName })),

    on(Actions.fetchEvents, (state) => ({ ...state, isLoadingEvents: true })),
    on(Actions.fetchEventsSuccess, (state, { data, page }) => ({
        ...state,
        isLoadingEvents: false,
        events: page === 1 ? privateEventsToModels(data) : [...state.events, ...privateEventsToModels(data)],
        eventsPage: data.events?.length > 0 ? page : state.eventsPage,
        totalEvents: data.total
    })),
    on(Actions.fetchEventsFailure, (state) => ({ ...state, isLoadingEvents: false })),

    on(Actions.fetchModuleEventsFilterItems, (state) => ({ ...state, isLoadingModuleEventsFilterItems: true })),
    on(Actions.fetchModuleEventsFilterItemsSuccess, (state, { agentNames, groupIds }) => ({
        ...state,
        isLoadingModuleEventsFilterItems: false,
        eventFilterItemAgentNames: agentNames,
        eventFilterItemGroupIds: groupIds
    })),
    on(Actions.fetchModuleEventsFilterItemsFailure, (state) => ({ ...state, isLoadingModuleEventsFilterItems: false })),

    on(Actions.setEventsGridFiltration, (state, { filtration }) => {
        const needRemoveFiltration =
            Array.isArray(filtration.value) && filtration.value.length === 1 && !filtration.value[0];
        const updatedFiltration = state.eventsGridFiltration.filter(
            (item: Filtration) => item.field !== filtration.field
        );

        return {
            ...state,
            eventsGridFiltration: [...updatedFiltration, ...(needRemoveFiltration ? [] : [filtration])]
        };
    }),
    on(Actions.setEventsGridSearch, (state, { value }) => ({ ...state, eventsGridSearch: value })),
    on(Actions.resetEventsFiltration, (state) => ({ ...state, eventsGridFiltration: [] })),
    on(Actions.setEventsGridSorting, (state, { sorting }) => ({ ...state, eventsSorting: sorting })),

    on(Actions.fetchModuleVersions, (state) => ({ ...state, isLoadingModuleVersions: true })),
    on(Actions.fetchModuleVersionSuccess, (state, { data }) => ({
        ...state,
        isLoadingModuleVersions: false,
        moduleVersions: data.modules
    })),
    on(Actions.fetchModuleVersionsFailed, (state) => ({ ...state, isLoadingModuleVersions: false })),

    on(Actions.enableModule, (state) => ({ ...state, isEnablingModule: true, enableModuleError: undefined })),
    on(Actions.enableModuleSuccess, (state) => ({ ...state, isEnablingModule: false })),
    on(Actions.enableModuleFailure, (state, { error }) => ({
        ...state,
        isEnablingModule: false,
        enableModuleError: error
    })),

    on(Actions.disableModule, (state) => ({ ...state, isDisablingModule: true, disableModuleError: undefined })),
    on(Actions.disableModuleSuccess, (state) => ({ ...state, isDisablingModule: false })),
    on(Actions.disableModuleFailure, (state, { error }) => ({
        ...state,
        isDisablingModule: false,
        disableModuleError: error
    })),

    on(Actions.deleteModule, (state) => ({ ...state, isDeletingModule: true, deleteModuleError: undefined })),
    on(Actions.deleteModuleSuccess, (state) => ({ ...state, isDeletingModule: false })),
    on(Actions.deleteModuleFailure, (state, { error }) => ({
        ...state,
        isDeletingModule: false,
        deleteModuleError: error
    })),

    on(Actions.changeModuleVersion, (state) => ({ ...state, isChangingVersion: true, changeVersionError: undefined })),
    on(Actions.changeModuleVersionSuccess, (state) => ({ ...state, isChangingVersion: false })),
    on(Actions.changeModuleVersionFailure, (state, { error }) => ({
        ...state,
        isChangingVersion: false,
        changeVersionError: error
    })),

    on(Actions.updateModule, (state) => ({ ...state, isUpdatingModule: true, updateModuleError: undefined })),
    on(Actions.updateModuleSuccess, (state) => ({ ...state, isUpdatingModule: false })),
    on(Actions.updateModuleFailure, (state, { error }) => ({
        ...state,
        isUpdatingModule: false,
        updateModuleError: error
    })),

    on(Actions.saveModuleConfig, (state) => ({ ...state, isSavingModuleConfig: true, savingModuleError: undefined })),
    on(Actions.saveModuleConfigSuccess, (state) => ({ ...state, isSavingModuleConfig: false })),
    on(Actions.saveModuleConfigFailure, (state, { error }) => ({
        ...state,
        isSavingModuleConfig: false,
        savingModuleError: error
    })),

    on(Actions.resetModuleErrors, (state) => ({
        ...state,
        changeVersionError: undefined,
        deleteModuleError: undefined,
        disableModuleError: undefined,
        enableModuleError: undefined,
        updateModuleError: undefined
    })),
    on(Actions.reset, () => ({ ...initialState }))
);
