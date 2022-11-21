import { Injectable } from '@angular/core';
import { Store } from '@ngrx/store';

import { ModelsModuleA } from '@soldr/api';
import { Filtration, Sorting, ViewMode } from '@soldr/shared';

import * as ModulesInstancesActions from './modules-instances.actions';
import { State } from './modules-instances.reducer';
import {
    selectChangeVersionError,
    selectDeleteModuleError,
    selectDisableModuleError,
    selectEnableModuleError,
    selectEntityId,
    selectEventFilterItemAgentIds,
    selectEventFilterItemGroupIds,
    selectEvents,
    selectEventsGridFiltration,
    selectEventsGridFiltrationByField,
    selectEventsGridSearch,
    selectEventsPage,
    selectIsChangingVersionModule,
    selectIsDeletingModule,
    selectIsDisablingModule,
    selectIsEnablingModule,
    selectIsLoadingEvents,
    selectIsLoadingModule,
    selectIsLoadingModulePolicy,
    selectIsLoadingModuleVersions,
    selectIsSavingModule,
    selectIsUpdatingModule,
    selectModule,
    selectModuleName,
    selectModulePolicy,
    selectModuleVersions,
    selectSavingModuleError,
    selectTotalEvents,
    selectUpdateModuleError,
    selectViewMode
} from './modules-instances.selectors';

@Injectable({
    providedIn: 'root'
})
export class ModulesInstancesFacade {
    changeVersionError$ = this.store.select(selectChangeVersionError);
    deleteError$ = this.store.select(selectDeleteModuleError);
    disableError$ = this.store.select(selectDisableModuleError);
    enableError$ = this.store.select(selectEnableModuleError);
    saveError$ = this.store.select(selectSavingModuleError);
    entityId$ = this.store.select(selectEntityId);
    events$ = this.store.select(selectEvents);
    eventsGridFiltration$ = this.store.select(selectEventsGridFiltration);
    eventsGridFiltrationByFields$ = this.store.select(selectEventsGridFiltrationByField);
    eventsPage$ = this.store.select(selectEventsPage);
    eventsSearchValue$ = this.store.select(selectEventsGridSearch);
    isChangingVersionModule$ = this.store.select(selectIsChangingVersionModule);
    isDeletingModule$ = this.store.select(selectIsDeletingModule);
    isEnablingModule$ = this.store.select(selectIsEnablingModule);
    isDisablingModule$ = this.store.select(selectIsDisablingModule);
    isLoadingEvents$ = this.store.select(selectIsLoadingEvents);
    isLoadingModule$ = this.store.select(selectIsLoadingModule);
    isLoadingModuleVersion$ = this.store.select(selectIsLoadingModuleVersions);
    isLoadingPolicy$ = this.store.select(selectIsLoadingModulePolicy);
    isSavingModule$ = this.store.select(selectIsSavingModule);
    isUpdatingModule$ = this.store.select(selectIsUpdatingModule);
    module$ = this.store.select(selectModule);
    moduleEventsFilterItemAgentIds$ = this.store.select(selectEventFilterItemAgentIds);
    moduleEventsFilterItemGroupIds$ = this.store.select(selectEventFilterItemGroupIds);
    moduleName$ = this.store.select(selectModuleName);
    moduleVersions$ = this.store.select(selectModuleVersions);
    policy$ = this.store.select(selectModulePolicy);
    totalEvents$ = this.store.select(selectTotalEvents);
    updateError$ = this.store.select(selectUpdateModuleError);
    viewMode$ = this.store.select(selectViewMode);

    constructor(private store: Store<State>) {}

    init(viewMode: ViewMode, entityId: number, moduleName: string) {
        this.store.dispatch(ModulesInstancesActions.init({ viewMode, entityId, moduleName }));
    }

    fetchModule(entityHash: string) {
        this.store.dispatch(ModulesInstancesActions.fetchModule({ entityHash }));
    }

    fetchEvents(page?: number): void {
        this.store.dispatch(ModulesInstancesActions.fetchEvents({ page }));
    }

    fetchModuleEventsFilterItems() {
        this.store.dispatch(ModulesInstancesActions.fetchModuleEventsFilterItems());
    }

    setEventsGridFiltration(filtration: Filtration): void {
        this.store.dispatch(ModulesInstancesActions.setEventsGridFiltration({ filtration }));
    }

    setEventsGridSearch(value: string): void {
        this.store.dispatch(ModulesInstancesActions.setEventsGridSearch({ value }));
    }

    resetEventsFiltration(): void {
        this.store.dispatch(ModulesInstancesActions.resetEventsFiltration());
    }

    setEventsGridSorting(sorting: Sorting): void {
        this.store.dispatch(ModulesInstancesActions.setEventsGridSorting({ sorting }));
    }

    fetchVersions(moduleName: string) {
        this.store.dispatch(ModulesInstancesActions.fetchModuleVersions({ moduleName }));
    }

    enableModule(policyHash: string, moduleName: string) {
        this.store.dispatch(ModulesInstancesActions.enableModule({ policyHash, moduleName }));
    }

    disableModule(policyHash: string, moduleName: string) {
        this.store.dispatch(ModulesInstancesActions.disableModule({ policyHash, moduleName }));
    }

    deleteModule(policyHash: string, moduleName: string) {
        this.store.dispatch(ModulesInstancesActions.deleteModule({ policyHash, moduleName }));
    }

    updateModule(policyHash: string, moduleName: string, version: string) {
        this.store.dispatch(ModulesInstancesActions.updateModule({ policyHash, moduleName, version }));
    }

    changeModuleVersion(policyHash: string, moduleName: string, version: string) {
        this.store.dispatch(ModulesInstancesActions.changeModuleVersion({ policyHash, moduleName, version }));
    }

    saveModuleConfig(policyHash: string, module: ModelsModuleA) {
        this.store.dispatch(ModulesInstancesActions.saveModuleConfig({ policyHash, module }));
    }

    resetModuleErrors() {
        this.store.dispatch(ModulesInstancesActions.resetModuleErrors());
    }

    reset() {
        this.store.dispatch(ModulesInstancesActions.reset());
    }
}
