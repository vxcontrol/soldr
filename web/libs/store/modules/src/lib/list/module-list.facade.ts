import { Injectable } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { Store } from '@ngrx/store';
import { map, Observable } from 'rxjs';

import { allListQuery, ModelsModuleInfo } from '@soldr/api';
import { UploadModule } from '@soldr/features/modules';
import { Module } from '@soldr/models';
import { Filtration, Sorting } from '@soldr/shared';

import * as ModulesActions from './module-list.actions';
import { State } from './module-list.reducer';
import {
    selectModules,
    selectTotal,
    selectIsLoadingModules,
    selectGridSorting,
    selectGridSearch,
    selectGridFiltration,
    selectPage,
    selectSelectedModules,
    selectGridFiltrationByField,
    selectIsCreatingModule,
    selectIsImportingModule,
    selectImportError,
    selectCreateError,
    selectCreatedModule,
    selectVersions,
    selectIsLoadingModuleVersions,
    selectRestored,
    selectIsDeletingModule,
    selectDeleteError,
    selectIsDeletingModuleVersion,
    selectDeleteVersionError,
    selectInitialListQuery,
    selectModulesTags
} from './module-list.selectors';

@Injectable({
    providedIn: 'root'
})
export class ModuleListFacade {
    allListQuery$ = this.store.select(selectInitialListQuery).pipe(map((initialQuery) => allListQuery(initialQuery)));
    createError$ = this.store.select(selectCreateError);
    createdModule$ = this.store.select(selectCreatedModule);
    deleteError$ = this.store.select(selectDeleteError);
    deleteVersionError$ = this.store.select(selectDeleteVersionError);
    gridFiltration$ = this.store.select(selectGridFiltration);
    gridFiltrationByField$ = this.store.select(selectGridFiltrationByField);
    importError$ = this.store.select(selectImportError);
    isCreatingModule$ = this.store.select(selectIsCreatingModule);
    isDeletingModule$ = this.store.select(selectIsDeletingModule);
    isDeletingModuleVersion$ = this.store.select(selectIsDeletingModuleVersion);
    isImportingModule$ = this.store.select(selectIsImportingModule);
    isLoading$ = this.store.select(selectIsLoadingModules);
    isLoadingModuleVersions$ = this.store.select(selectIsLoadingModuleVersions);
    isRestored$ = this.store.select(selectRestored);
    moduleVersions$ = this.store.select(selectVersions);
    modules$ = this.store.select(selectModules);
    modulesTags$ = this.store.select(selectModulesTags);
    page$ = this.store.select(selectPage);
    search$ = this.store.select(selectGridSearch);
    selectedModules$ = this.store.select(selectSelectedModules);
    sorting$ = this.store.select(selectGridSorting);
    total$ = this.store.select(selectTotal);

    constructor(private activatedRoute: ActivatedRoute, private store: Store<State>) {}

    fetchModules() {
        this.store.dispatch(ModulesActions.fetchModules());
    }

    fetchModulesPage(page?: number) {
        this.store.dispatch(ModulesActions.fetchModulesPage({ page }));
    }

    fetchModulesTags() {
        this.store.dispatch(ModulesActions.fetchModulesTags());
    }

    selectModules(modules: Module[]) {
        this.store.dispatch(ModulesActions.selectModules({ modules }));
    }

    setGridFiltration(filtration: Filtration) {
        this.store.dispatch(ModulesActions.setGridFiltration({ filtration }));
    }

    setGridFiltrationByTag(tag: string) {
        this.store.dispatch(ModulesActions.setGridFiltrationByTag({ tag }));
    }

    setGridSorting(sorting: Sorting) {
        this.store.dispatch(ModulesActions.setGridSorting({ sorting }));
    }

    setGridSearch(value: string) {
        this.store.dispatch(ModulesActions.setGridSearch({ value }));
    }

    createModule(module: ModelsModuleInfo) {
        this.store.dispatch(ModulesActions.createModule({ module }));
    }

    importModule(name: string, version: string, data: UploadModule) {
        this.store.dispatch(ModulesActions.importModule({ name, version, data }));
    }

    resetFiltration() {
        this.store.dispatch(ModulesActions.resetFiltration());
    }

    deleteModule(name: string) {
        this.store.dispatch(ModulesActions.deleteModule({ name }));
    }

    deleteModuleVersion(name: string, version: string) {
        this.store.dispatch(ModulesActions.deleteModuleVersion({ name, version }));
    }

    fetchModuleVersions(name: string) {
        this.store.dispatch(ModulesActions.fetchModuleVersions({ name }));
    }

    getIsModuleNameExists(name: string): Observable<boolean> {
        return this.store
            .select(selectModules)
            .pipe(map((modules) => modules.findIndex((module) => module.info.name === name) >= 0));
    }

    restoreState(): void {
        const params = this.activatedRoute.snapshot.queryParams as Record<string, string>;

        const gridFiltration: Filtration[] = params.filtration ? JSON.parse(params.filtration) : [];
        const gridSearch: string = params.search || '';
        const sorting: Sorting = params.sort ? JSON.parse(params.sort) : {};

        this.store.dispatch(ModulesActions.restoreState({ restoredState: { gridFiltration, gridSearch, sorting } }));
    }

    resetImportState() {
        this.store.dispatch(ModulesActions.resetImportState());
    }

    resetModuleErrors() {
        this.store.dispatch(ModulesActions.resetModuleErrors());
    }

    reset() {
        this.store.dispatch(ModulesActions.reset());
    }
}
