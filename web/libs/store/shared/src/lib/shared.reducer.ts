import { createReducer, on } from '@ngrx/store';

import {
    ErrorResponse,
    ModelsBinary,
    ModelsModuleS,
    ModelsOptionsActions,
    ModelsOptionsEvents,
    ModelsOptionsFields,
    ModelsOptionsTags,
    ModelsService,
    PublicInfo
} from '@soldr/api';
import {
    Agent,
    Group,
    manyAgentsToModels,
    manyModulesToModels,
    Policy,
    privateGroupsToModels,
    privatePoliciesToModels
} from '@soldr/models';

import * as SharedActions from './shared.actions';

export const sharedFeatureKey = 'shared';

export interface State {
    actions: ModelsOptionsActions[];
    agentBinaries: ModelsBinary[];
    allAgents: Agent[];
    allGroups: Group[];
    allModules: ModelsModuleS[];
    allPolicies: Policy[];
    allServices: ModelsService[];
    allTotalAgents: number;
    allTotalGroups: number;
    allTotalModules: number;
    allTotalPolicies: number;
    allTotalServices: number;
    events: ModelsOptionsEvents[];
    exportError: ErrorResponse;
    fields: ModelsOptionsFields[];
    info?: PublicInfo;
    initializedGroups: boolean;
    isChangingPassword: boolean;
    isExportingBinaryFile: boolean;
    isInfoLoaded: boolean;
    isInfoLoading: boolean;
    isLoadingAgentBinaries: boolean;
    isLoadingAllAgents: boolean;
    isLoadingAllGroups: boolean;
    isLoadingAllModules: boolean;
    isLoadingAllPolicies: boolean;
    isLoadingAllServices: boolean;
    isLoadingLatestAgentBinary: boolean;
    isLoadingOptionsActions: boolean;
    isLoadingOptionsEvents: boolean;
    isLoadingOptionsFields: boolean;
    isLoadingOptionsTags: boolean;
    latestAgentBinary: ModelsBinary;
    passwordChangeError: ErrorResponse;
    searchValue: string;
    selectedTags: string[];
    tags: ModelsOptionsTags[];
}

export const initialState: State = {
    actions: [],
    agentBinaries: undefined,
    allAgents: [],
    allGroups: [],
    allModules: [],
    allPolicies: [],
    allServices: [],
    allTotalAgents: 0,
    allTotalGroups: 0,
    allTotalModules: 0,
    allTotalPolicies: 0,
    allTotalServices: 0,
    events: [],
    exportError: undefined,
    fields: [],
    info: undefined,
    initializedGroups: false,
    isChangingPassword: false,
    isExportingBinaryFile: false,
    isInfoLoaded: false,
    isInfoLoading: false,
    isLoadingAgentBinaries: false,
    isLoadingAllAgents: false,
    isLoadingAllGroups: false,
    isLoadingAllModules: false,
    isLoadingAllPolicies: false,
    isLoadingAllServices: false,
    isLoadingLatestAgentBinary: false,
    isLoadingOptionsActions: false,
    isLoadingOptionsEvents: false,
    isLoadingOptionsFields: false,
    isLoadingOptionsTags: false,
    latestAgentBinary: undefined,
    passwordChangeError: undefined,
    searchValue: '',
    selectedTags: [],
    tags: []
};

export const reducer = createReducer(
    initialState,

    on(SharedActions.changePassword, (state) => ({
        ...state,
        isChangingPassword: true,
        passwordChangeError: undefined
    })),
    on(SharedActions.changePasswordSuccess, (state) => ({ ...state, isChangingPassword: false })),
    on(SharedActions.changePasswordFailure, (state, { error }) => ({
        ...state,
        isChangingPassword: false,
        passwordChangeError: error
    })),

    on(SharedActions.fetchAllAgents, (state) => ({ ...state, isLoadingAllAgents: true })),
    on(SharedActions.fetchAllAgentsSuccess, (state, { data }) => ({
        ...state,
        isLoadingAllAgents: false,
        allTotalAgents: data.total,
        allAgents: manyAgentsToModels(data)
    })),
    on(SharedActions.fetchAllAgentsFailure, (state) => ({ ...state, isLoadingAllAgents: false })),

    on(SharedActions.fetchAllGroups, (state, { silent }) => ({ ...state, isLoadingAllGroups: !silent })),
    on(SharedActions.fetchAllGroupsSuccess, (state, { data }) => ({
        ...state,
        isLoadingAllGroups: false,
        initializedGroups: true,
        allTotalGroups: data.total,
        allGroups: privateGroupsToModels(data)
    })),
    on(SharedActions.fetchAllGroupsFailure, (state) => ({ ...state, isLoadingAllGroups: false })),

    on(SharedActions.fetchAllPolicies, (state, { silent }) => ({ ...state, isLoadingAllPolicies: !silent })),
    on(SharedActions.fetchAllPoliciesSuccess, (state, { data }) => ({
        ...state,
        isLoadingAllPolicies: false,
        allPolicies: privatePoliciesToModels(data),
        allTotal: data.total
    })),
    on(SharedActions.fetchAllPoliciesFailure, (state) => ({ ...state, isLoadingAllPolicies: false })),

    on(SharedActions.fetchAllServices, (state: State) => ({ ...state, isLoadingAllServices: true })),
    on(SharedActions.fetchAllServicesSuccess, (state: State, { data }) => ({
        ...state,
        allServices: data.services,
        allTotalServices: data.total,
        isLoadingAllServices: false
    })),
    on(SharedActions.fetchAllServicesFailure, (state: State) => ({ ...state, isLoadingAllServices: false })),

    on(SharedActions.fetchAllModules, (state) => ({ ...state, isLoadingAllModules: true })),
    on(SharedActions.fetchAllModulesSuccess, (state, { data }) => ({
        ...state,
        isLoadingAllModules: false,
        allTotalModules: data.total,
        allModules: manyModulesToModels(data)
    })),
    on(SharedActions.fetchAllModulesFailure, (state) => ({ ...state, isLoadingAllModules: false })),

    on(SharedActions.exportBinaryFile, (state) => ({
        ...state,
        isExportingBinaryFile: true,
        exportError: undefined
    })),
    on(SharedActions.exportBinaryFileSuccess, (state: State) => ({ ...state, isExportingBinaryFile: false })),
    on(SharedActions.exportBinaryFileFailure, (state: State, { error }) => ({
        ...state,
        isExportingBinaryFile: false,
        exportError: error
    })),

    on(SharedActions.fetchInfo, (state: State) => ({ ...state, isInfoLoading: true, isInfoLoaded: false })),
    on(SharedActions.fetchInfoSuccess, (state: State, { info }) => ({
        ...state,
        info,
        isInfoLoading: false,
        isInfoLoaded: true
    })),
    on(SharedActions.fetchInfoFailure, (state: State) => ({ ...state, isInfoLoading: false, isInfoLoaded: false })),

    on(SharedActions.fetchLatestAgentBinary, (state) => ({ ...state, isLoadingLatestAgentBinary: true })),
    on(SharedActions.fetchLatestAgentBinarySuccess, (state: State, { binaries }) => ({
        ...state,
        latestAgentBinary: binaries.binaries[0],
        isLoadingLatestAgentBinary: false
    })),
    on(SharedActions.fetchLatestAgentBinaryFailure, (state) => ({ ...state, isLoadingLatestAgentBinary: false })),

    on(SharedActions.fetchAgentBinaries, (state) => ({ ...state, isLoadingAgentBinaries: true })),
    on(SharedActions.fetchAgentBinariesSuccess, (state: State, { binaries }) => ({
        ...state,
        agentBinaries: [...binaries.binaries],
        latestAgentBinary: binaries.binaries[0],
        isLoadingAgentBinaries: false
    })),
    on(SharedActions.fetchAgentBinariesFailure, (state) => ({ ...state, isLoadingAgentBinaries: false })),

    on(SharedActions.logout, (state: State) => ({ ...state, info: undefined as PublicInfo })),

    on(SharedActions.fetchOptionsActions, (state: State) => ({ ...state, isLoadingOptionsActions: true })),
    on(SharedActions.fetchOptionsActionsSuccess, (state: State, { data }) => ({
        ...state,
        isLoadingOptionsActions: false,
        actions: data.actions
    })),
    on(SharedActions.fetchOptionsActionsFailure, (state: State) => ({ ...state, isLoadingOptionsActions: false })),

    on(SharedActions.fetchOptionsEvents, (state: State) => ({ ...state, isLoadingOptionsEvents: true })),
    on(SharedActions.fetchOptionsEventsSuccess, (state: State, { data }) => ({
        ...state,
        isLoadingOptionsEvents: false,
        events: data.events
    })),
    on(SharedActions.fetchOptionsEventsFailure, (state: State) => ({ ...state, isLoadingOptionsEvents: false })),

    on(SharedActions.fetchOptionsFields, (state: State) => ({ ...state, isLoadingOptionsFields: true })),
    on(SharedActions.fetchOptionsFieldsSuccess, (state: State, { data }) => ({
        ...state,
        isLoadingOptionsFields: false,
        fields: data.fields
    })),
    on(SharedActions.fetchOptionsFieldsFailure, (state: State) => ({ ...state, isLoadingOptionsFields: false })),

    on(SharedActions.fetchOptionsTags, (state: State) => ({ ...state, isLoadingOptionsTags: true })),
    on(SharedActions.fetchOptionsTagsSuccess, (state: State, { data }) => ({
        ...state,
        isLoadingOptionsTags: false,
        tags: data.tags
    })),
    on(SharedActions.fetchOptionsTagsFailure, (state: State) => ({ ...state, isLoadingOptionsTags: false })),
    on(SharedActions.setFilterByTags, (state: State, { tags }) => ({ ...state, selectedTags: tags })),
    on(SharedActions.resetFilterByTags, (state: State) => ({ ...state, selectedTags: [] as string[] })),
    on(SharedActions.setFilterBySearchValue, (state: State, { searchValue }) => ({ ...state, searchValue }))
);
