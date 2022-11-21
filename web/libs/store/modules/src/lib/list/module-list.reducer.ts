import { createReducer, on } from '@ngrx/store';

import { ErrorResponse, ModelsModuleS, ModelsModuleSShort } from '@soldr/api';
import { manyModulesToModels } from '@soldr/models';
import { Filtration, Sorting } from '@soldr/shared';

import * as ModulesActions from './module-list.actions';

export const moduleListFeatureKey = 'module-list';

export interface State {
    createError: ErrorResponse;
    createdModule: ModelsModuleS;
    deleteError: ErrorResponse;
    deleteVersionError: ErrorResponse;
    gridFiltration: Filtration[];
    gridSearch: string;
    importError: ErrorResponse;
    isCreatingModule: boolean;
    isDeleteNodule: boolean;
    isDeleteNoduleVersion: boolean;
    isImportingModule: boolean;
    isLoadingModule: boolean;
    isLoadingModules: boolean;
    isLoadingModulesTags: boolean;
    isLoadingVersions: boolean;
    module: ModelsModuleS;
    modules: ModelsModuleS[];
    modulesTags: string[];
    page: number;
    restored: boolean;
    selectedIds: number[];
    sorting: Sorting | Record<never, any>;
    total: number;
    versions: ModelsModuleSShort[];
}

export const initialState: State = {
    createError: undefined,
    createdModule: undefined,
    deleteError: undefined,
    deleteVersionError: undefined,
    gridFiltration: [],
    gridSearch: undefined,
    importError: undefined,
    isCreatingModule: false,
    isDeleteNodule: false,
    isDeleteNoduleVersion: false,
    isImportingModule: false,
    isLoadingModule: false,
    isLoadingModules: false,
    isLoadingModulesTags: false,
    isLoadingVersions: false,
    module: undefined,
    modules: [],
    modulesTags: [],
    page: undefined,
    restored: false,
    selectedIds: [],
    sorting: {},
    total: 0,
    versions: []
};

export const reducer = createReducer(
    initialState,

    on(ModulesActions.fetchModulesPage, (state) => ({ ...state, isLoadingModules: true })),
    on(ModulesActions.fetchModulesSuccess, (state, { data }) => ({
        ...state,
        isLoadingModules: false,
        modules: manyModulesToModels(data),
        total: data.total
    })),
    on(ModulesActions.fetchModulesPage, (state) => ({ ...state, isLoadingModules: false })),

    on(ModulesActions.fetchModulesPage, (state) => ({ ...state, isLoadingModules: true })),
    on(ModulesActions.fetchModulesPageSuccess, (state, { data, page }) => ({
        ...state,
        isLoadingModules: false,
        isInitializedModule: true,
        modules: page === 1 ? manyModulesToModels(data) : [...state.modules, ...manyModulesToModels(data)],
        page: data.modules?.length > 0 ? page : state.page,
        total: data.total
    })),
    on(ModulesActions.fetchModulesPageFailure, (state) => ({ ...state, isLoadingModules: false })),

    on(ModulesActions.fetchModulesTags, (state) => ({ ...state, isLoadingModulesTags: true })),
    on(ModulesActions.fetchModulesTagsSuccess, (state, { tags }) => ({
        ...state,
        isLoadingModulesTags: false,
        modulesTags: tags
    })),
    on(ModulesActions.fetchModulesTagsFailure, (state) => ({ ...state, isLoadingModulesTags: false })),

    on(ModulesActions.createModule, (state) => ({ ...state, isCreatingModule: true, createError: undefined })),
    on(ModulesActions.createModuleSuccess, (state, { module }) => ({
        ...state,
        isCreatingModule: false,
        createdModule: module
    })),
    on(ModulesActions.createModuleFailure, (state, { error }) => ({
        ...state,
        isCreatingModule: false,
        createError: error,
        createdModule: undefined
    })),

    on(ModulesActions.importModule, (state) => ({ ...state, isImportingModule: true, importError: undefined })),
    on(ModulesActions.importModuleSuccess, (state) => ({ ...state, isImportingModule: false })),
    on(ModulesActions.importModuleFailure, (state, { error }) => ({
        ...state,
        isImportingModule: false,
        importError: error
    })),

    on(ModulesActions.selectModules, (state, { modules }) => ({ ...state, selectedIds: modules.map(({ id }) => id) })),
    on(ModulesActions.selectModulesByIds, (state, { ids }) => ({ ...state, selectedIds: ids })),

    on(ModulesActions.setGridFiltration, (state, { filtration }) => {
        const needRemoveFiltration =
            Array.isArray(filtration.value) && filtration.value.length === 1 && !filtration.value[0];
        const updatedFiltration = state.gridFiltration.filter((item: Filtration) => item.field !== filtration.field);

        return { ...state, gridFiltration: [...updatedFiltration, ...(needRemoveFiltration ? [] : [filtration])] };
    }),
    on(ModulesActions.setGridFiltrationByTag, (state, { tag }) => {
        const updatedFiltration = state.gridFiltration.filter((item: Filtration) => item.field !== 'tags');

        return { ...state, gridFiltration: [...updatedFiltration, { field: 'tags', value: [tag] }] };
    }),
    on(ModulesActions.setGridSearch, (state, { value }) => ({ ...state, gridSearch: value })),
    on(ModulesActions.resetFiltration, (state) => ({ ...state, gridFiltration: [] })),
    on(ModulesActions.setGridSorting, (state, { sorting }) => ({ ...state, sorting })),

    on(ModulesActions.deleteModule, (state) => ({ ...state, isDeleteNodule: true, deleteError: undefined })),
    on(ModulesActions.deleteModuleSuccess, (state) => ({ ...state, isDeleteNodule: false })),
    on(ModulesActions.deleteModuleFailure, (state, { error }) => ({
        ...state,
        isDeleteNodule: false,
        deleteError: error
    })),

    on(ModulesActions.deleteModuleVersion, (state) => ({
        ...state,
        isDeleteNoduleVersion: true,
        deleteVersionError: undefined
    })),
    on(ModulesActions.deleteModuleVersionSuccess, (state) => ({ ...state, isDeleteNoduleVersion: false })),
    on(ModulesActions.deleteModuleVersionFailure, (state, { error }) => ({
        ...state,
        isDeleteNoduleVersion: false,
        deleteVersionError: error
    })),

    on(ModulesActions.fetchModuleVersions, (state) => ({ ...state, isLoadingVersions: true })),
    on(ModulesActions.fetchModuleVersionsSuccess, (state, { data }) => ({
        ...state,
        isLoadingVersions: false,
        versions: data.modules
    })),
    on(ModulesActions.fetchModuleVersionsFailure, (state) => ({ ...state, isLoadingVersions: false })),

    on(ModulesActions.restoreState, (state, { restoredState }) => ({ ...state, restored: true, ...restoredState })),

    on(ModulesActions.resetImportState, (state) => ({ ...state, importError: undefined })),
    on(ModulesActions.resetModuleErrors, (state) => ({
        ...state,
        createError: undefined,
        deleteError: undefined,
        exportError: undefined,
        importError: undefined,
        deleteVersionError: undefined
    })),
    on(ModulesActions.reset, () => ({ ...initialState }))
);
