import { Injectable } from '@angular/core';
import { Store } from '@ngrx/store';
import { filter, map, Observable } from 'rxjs';

import { ModelsBinary, ModelsPassword, PublicInfo } from '@soldr/api';
import { Architecture, OperationSystem } from '@soldr/shared';

import * as SharedActions from './shared.actions';
import { State } from './shared.reducer';
import * as SharedSelectors from './shared.selectors';
import { selectAllPolicies, selectAllTotalPolicies, selectIsLoadingAllPolicies } from './shared.selectors';

@Injectable({
    providedIn: 'root'
})
export class SharedFacade {
    agentBinaries$ = this.store.select(SharedSelectors.selectBinaries);
    agentBinaryVersions$ = this.agentBinaries$.pipe(
        map((agentBinaries: ModelsBinary[]) => agentBinaries?.map((binary) => binary.version))
    );
    allAgents$ = this.store.select(SharedSelectors.selectAllAgents);
    allGroups$ = this.store.select(SharedSelectors.selectAllGroups);
    allModules$ = this.store.select(SharedSelectors.selectAllModules);
    allPolicies$ = this.store.select(selectAllPolicies);
    allServices$ = this.store.select(SharedSelectors.selectAllServices);
    allTotalAgents$ = this.store.select(SharedSelectors.selectAllTotalAgents);
    allTotalModules$ = this.store.select(SharedSelectors.selectAllTotalModules);
    allTotalPolicies$ = this.store.select(selectAllTotalPolicies);
    isChangingPassword$ = this.store.select(SharedSelectors.selectIsChangingPassword);
    isExportingBinaryFile$ = this.store.select(SharedSelectors.selectIsExportingBinaryFile);
    exportError$ = this.store.select(SharedSelectors.selectExportError);
    initializedGroups$ = this.store.select(SharedSelectors.selectInitializedGroups);
    isPasswordChangeRequired$ = this.store.select(SharedSelectors.selectIsPasswordChangeRequired);
    isLoadingActions$ = this.store.select(SharedSelectors.selectIsLoadingActions);
    isLoadingAgentBinaries$ = this.store.select(SharedSelectors.selectIsLoadingBinaries);
    isLoadingAllGroups$ = this.store.select(SharedSelectors.selectIsLoadingAllGroups);
    isLoadingAllModules$ = this.store.select(SharedSelectors.selectIsLoadingAllModules);
    isLoadingAllPolicies$ = this.store.select(selectIsLoadingAllPolicies);
    isLoadingAllServices$ = this.store.select(SharedSelectors.selectIsLoadingAllServices);
    isLoadingEvents$ = this.store.select(SharedSelectors.selectIsLoadingEvents);
    isLoadingFields$ = this.store.select(SharedSelectors.selectIsLoadingFields);
    isLoadingLatestAgentBinary$ = this.store.select(SharedSelectors.selectIsLoadingLatestBinary);
    isLoadingTags$ = this.store.select(SharedSelectors.selectIsLoadingTags);
    latestAgentBinary$ = this.store.select(SharedSelectors.selectLatestBinary);
    latestAgentBinaryVersion$ = this.latestAgentBinary$.pipe(map((agentBinary: ModelsBinary) => agentBinary?.version));
    passwordChangeError$ = this.store.select(SharedSelectors.selectPasswordChangeError);
    selectedServiceName$ = this.selectInfo().pipe(map((info) => info?.service?.name || ''));
    selectedServiceUrl$ = this.selectInfo().pipe(
        filter((info) => !!info?.service),
        map(({ service }) => `/services/${service?.hash}`)
    );
    shortServices$ = this.store.select(SharedSelectors.selectShortServices);
    optionsActions$ = this.store.select(SharedSelectors.selectActions);
    optionsEvents$ = this.store.select(SharedSelectors.selectEvents);
    optionsFields$ = this.store.select(SharedSelectors.selectFields);
    optionsTags$ = this.store.select(SharedSelectors.selectTags);
    selectedGroupTags$ = this.store.select(SharedSelectors.selectSelectedTags);
    searchValue$ = this.store.select(SharedSelectors.selectSearchValue);

    constructor(private store: Store<State>) {}

    changePassword(data: ModelsPassword) {
        this.store.dispatch(SharedActions.changePassword({ data }));
    }

    exportBinary(os: OperationSystem, arch: Architecture, version: string): void {
        this.store.dispatch(SharedActions.exportBinaryFile({ os, arch, version }));
    }

    fetchAllAgents() {
        this.store.dispatch(SharedActions.fetchAllAgents());
    }

    fetchAllGroups(silent = false) {
        this.store.dispatch(SharedActions.fetchAllGroups({ silent }));
    }

    fetchAllPolicies(silent = false) {
        this.store.dispatch(SharedActions.fetchAllPolicies({ silent }));
    }

    fetchAllServices() {
        this.store.dispatch(SharedActions.fetchAllServices());
    }

    fetchAllModules() {
        this.store.dispatch(SharedActions.fetchAllModules());
    }

    fetchInfo(refreshCookie = true): void {
        this.store.dispatch(SharedActions.fetchInfo({ refreshCookie }));
    }

    fetchAgentBinaries(): void {
        this.store.dispatch(SharedActions.fetchAgentBinaries());
    }

    fetchLatestAgentBinary(): void {
        this.store.dispatch(SharedActions.fetchLatestAgentBinary());
    }

    fetchActions(): void {
        this.store.dispatch(SharedActions.fetchOptionsActions());
    }

    fetchEvents(): void {
        this.store.dispatch(SharedActions.fetchOptionsEvents());
    }

    fetchFields(): void {
        this.store.dispatch(SharedActions.fetchOptionsFields());
    }

    fetchTags(): void {
        this.store.dispatch(SharedActions.fetchOptionsTags());
    }

    selectInfo(): Observable<PublicInfo | undefined> {
        return this.store.select(SharedSelectors.selectInfo);
    }

    selectIsInfoLoaded(): Observable<boolean> {
        return this.store.select(SharedSelectors.selectIsInfoLoaded);
    }

    selectIsAuthorized(): Observable<boolean> {
        return this.store.select(SharedSelectors.selectIsAuthorized);
    }

    setFilterByTags(tags: string[]) {
        this.store.dispatch(SharedActions.setFilterByTags({ tags }));
    }

    resetFilterByTags() {
        this.store.dispatch(SharedActions.resetFilterByTags());
    }

    setSearchValue(searchValue: string) {
        this.store.dispatch(SharedActions.setFilterBySearchValue({ searchValue }));
    }

    logout(): void {
        document.cookie = 'auth=; Path=/; Expires=Thu, 01 Jan 1970 00:00:01 GMT;';
        this.store.dispatch(SharedActions.logout());
    }
}
