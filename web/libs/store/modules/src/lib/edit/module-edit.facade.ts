import { Injectable } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { Store } from '@ngrx/store';
import { map, shareReplay, take } from 'rxjs';

import {
    ModelsChangelog,
    ModelsChangelogVersion,
    ModelsDependencyItem,
    ModelsLocale,
    ModelsModuleInfo,
    ModelsModuleS
} from '@soldr/api';
import { ModuleVersionPipe, NcformSchema } from '@soldr/shared';

import * as ModuleEditActions from './module-edit.actions';
import { State, ValidationState } from './module-edit.reducer';
import {
    selectActionsConfigSection,
    selectAgentVersionDependency,
    selectAgentVersions,
    selectAllModules,
    selectChangedActions,
    selectChangedEvents,
    selectChangedSecureParams,
    selectChangelog,
    selectConfigSection,
    selectCreateDraftError,
    selectDeleteError,
    selectDeleteVersionError,
    selectEventsConfigSection,
    selectFetchFilesError,
    selectFieldsConfigSection,
    selectFiles,
    selectIsCreatingDraft,
    selectIsDeletingModule,
    selectIsDeletingModuleVersion,
    selectIsDirty,
    selectIsDirtyConfig,
    selectIsDirtyGeneral,
    selectIsFetchAllModules,
    selectIsFetchFiles,
    selectIsLoadingFile,
    selectIsLoadingModule,
    selectIsLoadingModuleVersions,
    selectIsReleasingModule,
    selectIsSavingFiles,
    selectIsSavingModule,
    selectIsUpdatingModuleInPolicies,
    selectIsValidActions,
    selectIsValidChangelog,
    selectIsValidConfiguration,
    selectIsValidDependencies,
    selectIsValidEvents,
    selectIsValidFields,
    selectIsValidGeneral,
    selectIsValidLocalization,
    selectIsValidSecureConfiguration,
    selectLoadFileErrors,
    selectLocalizationSection,
    selectModule,
    selectModuleVersionsByName,
    selectOpenedFiles,
    selectReceiveDataDependencies,
    selectReleaseError,
    selectRestored,
    selectSaveFileErrors,
    selectSecureConfigSection,
    selectSendDataDependencies,
    selectUnusedFields,
    selectUnusedRequiredFields,
    selectUpdateModuleInPoliciesError,
    selectUpdates,
    selectVersions
} from './module-edit.selectors';

@Injectable({
    providedIn: 'root'
})
export class ModuleEditFacade {
    actionsConfigSchemaModel$ = this.store.select(selectActionsConfigSection);
    agentVersionDependency$ = this.store.select(selectAgentVersionDependency);
    agentVersions$ = this.store.select(selectAgentVersions);
    allModules$ = this.store.select(selectAllModules);
    canUpdateModuleInPolicies$ = this.store.select(selectUpdates).pipe(map((data) => data?.policies?.length > 0));
    changedActions$ = this.store.select(selectChangedActions);
    changedEvents$ = this.store.select(selectChangedEvents);
    changedSecureParams$ = this.store.select(selectChangedSecureParams);
    changelog$ = this.store.select(selectChangelog);
    configSchemaModel$ = this.store.select(selectConfigSection);
    createDraftError$ = this.store.select(selectCreateDraftError);
    deleteError$ = this.store.select(selectDeleteError);
    deleteVersionError$ = this.store.select(selectDeleteVersionError);
    eventsConfigSchemaModel$ = this.store.select(selectEventsConfigSection);
    fetchFilesError$ = this.store.select(selectFetchFilesError);
    fieldsSchemaModel$ = this.store.select(selectFieldsConfigSection);
    files$ = this.store.select(selectFiles);
    isCreatingDraft$ = this.store.select(selectIsCreatingDraft);
    isDeletingModule$ = this.store.select(selectIsDeletingModule);
    isDeletingModuleVersion$ = this.store.select(selectIsDeletingModuleVersion);
    isDirty$ = this.store.select(selectIsDirty);
    isDirtyConfig$ = this.store.select(selectIsDirtyConfig);
    isDirtyGeneral$ = this.store.select(selectIsDirtyGeneral);
    isFetchAllModules$ = this.store.select(selectIsFetchAllModules);
    isFetchFiles$ = this.store.select(selectIsFetchFiles);
    isLoadingFiles$ = this.store.select(selectIsLoadingFile);
    isLoadingModule$ = this.store.select(selectIsLoadingModule);
    isLoadingModuleVersions$ = this.store.select(selectIsLoadingModuleVersions);
    isReleasingModule$ = this.store.select(selectIsReleasingModule);
    isRestored$ = this.store.select(selectRestored);
    isSavingFiles$ = this.store.select(selectIsSavingFiles);
    isValidConfiguration$ = this.store.select(selectIsValidConfiguration);
    isValidGeneral$ = this.store.select(selectIsValidGeneral);
    isValidSecureConfiguration$ = this.store.select(selectIsValidSecureConfiguration);
    isValidEvents$ = this.store.select(selectIsValidEvents);
    isValidActions$ = this.store.select(selectIsValidActions);
    isValidFields$ = this.store.select(selectIsValidFields);
    isValidDependencies$ = this.store.select(selectIsValidDependencies);
    isValidLocalization$ = this.store.select(selectIsValidLocalization);
    isValidFiles$ = this.store.select(selectIsValidFields);
    isValidChangelog$ = this.store.select(selectIsValidChangelog);
    isSavingModule$ = this.store.select(selectIsSavingModule);
    isUpdatingModuleInPolicies$ = this.store.select(selectIsUpdatingModuleInPolicies);
    loadFileErrors$ = this.store.select(selectLoadFileErrors);
    localizationModel$ = this.store.select(selectLocalizationSection);
    module$ = this.store.select(selectModule);
    moduleVersions$ = this.store.select(selectVersions);
    moduleVersionsByName$ = this.store.select(selectModuleVersionsByName);
    openedFiles$ = this.store.select(selectOpenedFiles);
    receiveDataDependencies$ = this.store.select(selectReceiveDataDependencies);
    releaseError$ = this.store.select(selectReleaseError);
    saveFileErrors$ = this.store.select(selectSaveFileErrors);
    secureConfigSchemaModel$ = this.store.select(selectSecureConfigSection);
    sendDataDependencies$ = this.store.select(selectSendDataDependencies);
    updateModuleInPoliciesError$ = this.store.select(selectUpdateModuleInPoliciesError);
    unusedFields$ = this.store.select(selectUnusedFields);
    unusedRequiredFields$ = this.store.select(selectUnusedRequiredFields);

    constructor(private activatedRoute: ActivatedRoute, private store: Store<State>) {}

    // LOADING
    fetchModule(name: string, version: string = 'latest') {
        this.store.dispatch(ModuleEditActions.fetchModule({ name, version }));
    }

    fetchModuleVersions(name: string) {
        this.store.dispatch(ModuleEditActions.fetchModuleVersions({ name }));
    }

    fetchUpdates(name: string, version: string = 'latest') {
        this.store.dispatch(ModuleEditActions.fetchModuleUpdates({ name, version }));
    }

    // OPERATIONS
    saveModule() {
        this.store.dispatch(ModuleEditActions.saveModule());
    }

    releaseModule(record: ModelsChangelogVersion) {
        this.module$
            .pipe(shareReplay({ bufferSize: 1, refCount: false }), take(1))
            .subscribe((module: ModelsModuleS) => {
                const name = module.info.name;
                const version = new ModuleVersionPipe().transform(module.info.version);
                const releasedModule = {
                    ...module,
                    changelog: {
                        ...module.changelog,
                        [version]: record
                    }
                };
                this.store.dispatch(ModuleEditActions.releaseModule({ name, version, module: releasedModule }));
            });
    }

    createDraft(version: string, record: ModelsChangelogVersion) {
        this.module$
            .pipe(shareReplay({ bufferSize: 1, refCount: false }), take(1))
            .subscribe((module: ModelsModuleS) => {
                const name = module.info.name;
                this.store.dispatch(ModuleEditActions.createModuleDraft({ name, version, changelog: record }));
            });
    }

    deleteModule(name: string) {
        this.store.dispatch(ModuleEditActions.deleteModule({ name }));
    }

    deleteModuleVersion(name: string, version: string) {
        this.store.dispatch(ModuleEditActions.deleteModuleVersion({ name, version }));
    }

    updateModuleInPolicies(name: string, version: string) {
        this.store.dispatch(ModuleEditActions.updateModuleInPolicies({ name, version }));
    }

    // GENERAL
    updateGeneralSection(info: ModelsModuleInfo) {
        this.store.dispatch(ModuleEditActions.updateGeneralSection({ info }));
    }

    // CONFIG
    addConfigParam() {
        this.store.dispatch(ModuleEditActions.addConfigParam());
    }

    removeConfigParam(name: string) {
        this.store.dispatch(ModuleEditActions.removeConfigParam({ name }));
    }

    removeAllConfigParams() {
        this.store.dispatch(ModuleEditActions.removeAllConfigParams());
    }

    updateConfigSchema(schema: NcformSchema) {
        this.store.dispatch(ModuleEditActions.updateConfigSchema({ schema }));
    }

    updateDefaultConfig(defaultConfig: Record<string, any>) {
        this.store.dispatch(ModuleEditActions.updateDefaultConfig({ defaultConfig }));
    }

    // SECURE CONFIG
    addSecureConfigParam() {
        this.store.dispatch(ModuleEditActions.addSecureConfigParam());
    }

    removeSecureConfigParam(name: string) {
        this.store.dispatch(ModuleEditActions.removeSecureConfigParam({ name }));
    }

    removeAllSecureConfigParams() {
        this.store.dispatch(ModuleEditActions.removeAllSecureConfigParams());
    }

    updateSecureConfigSchema(schema: NcformSchema) {
        this.store.dispatch(ModuleEditActions.updateSecureConfigSchema({ schema }));
    }

    updateSecureDefaultConfig(defaultConfig: Record<string, any>) {
        this.store.dispatch(ModuleEditActions.updateSecureDefaultConfig({ defaultConfig }));
    }

    // EVENTS
    addEvent() {
        this.store.dispatch(ModuleEditActions.addEvent());
    }

    removeEvent(name: string) {
        this.store.dispatch(ModuleEditActions.removeEvent({ name }));
    }

    removeAllEvents() {
        this.store.dispatch(ModuleEditActions.removeAllEvents());
    }

    updateEventsSchema(schema: NcformSchema) {
        this.store.dispatch(ModuleEditActions.updateEventsSchema({ schema }));
    }

    updateEventsDefaultConfig(defaultConfig: Record<string, any>) {
        this.store.dispatch(ModuleEditActions.updateEventsDefaultConfig({ defaultConfig }));
    }

    addEventKey(eventName: string) {
        this.store.dispatch(ModuleEditActions.addEventKey({ eventName }));
    }

    removeEventKey(eventName: string, keyName: string) {
        this.store.dispatch(ModuleEditActions.removeEventKey({ eventName, keyName }));
    }

    // ACTIONS
    addAction() {
        this.store.dispatch(ModuleEditActions.addAction());
    }

    removeAction(name: string) {
        this.store.dispatch(ModuleEditActions.removeAction({ name }));
    }

    removeAllActions() {
        this.store.dispatch(ModuleEditActions.removeAllActions());
    }

    updateActionsSchema(schema: NcformSchema) {
        this.store.dispatch(ModuleEditActions.updateActionsSchema({ schema }));
    }

    updateActionsDefaultConfig(defaultConfig: Record<string, any>) {
        this.store.dispatch(ModuleEditActions.updateActionsDefaultConfig({ defaultConfig }));
    }

    addActionKey(actionName: string) {
        this.store.dispatch(ModuleEditActions.addActionKey({ actionName }));
    }

    removeActionKey(actionName: string, keyName: string) {
        this.store.dispatch(ModuleEditActions.removeActionKey({ actionName, keyName }));
    }

    // FIELDS
    addField() {
        this.store.dispatch(ModuleEditActions.addField());
    }

    removeField(name: string) {
        this.store.dispatch(ModuleEditActions.removeField({ name }));
    }

    removeAllFields() {
        this.store.dispatch(ModuleEditActions.removeAllFields());
    }

    updateFieldsSchema(schema: NcformSchema) {
        this.store.dispatch(ModuleEditActions.updateFieldsSchema({ schema }));
    }

    // LOCALIZATION
    updateLocalizationModel(model: Partial<ModelsLocale>) {
        this.store.dispatch(ModuleEditActions.updateLocalizationModel({ model }));
    }

    // FILES
    fetchFiles(name: string, version: string = 'latest') {
        this.store.dispatch(ModuleEditActions.fetchFiles({ name, version }));
    }

    loadFiles(paths: string[]) {
        this.module$.pipe(shareReplay({ bufferSize: 1, refCount: false }), take(1)).subscribe((module) => {
            const moduleName = module.info.name;
            const version = new ModuleVersionPipe().transform(module.info.version);

            this.store.dispatch(ModuleEditActions.loadFiles({ moduleName, version, paths }));
        });
    }

    saveFiles(files: { path: string; content: string }[]) {
        this.module$.pipe(shareReplay({ bufferSize: 1, refCount: false }), take(1)).subscribe((module) => {
            const moduleName = module.info.name;
            const version = new ModuleVersionPipe().transform(module.info.version);

            this.store.dispatch(ModuleEditActions.saveFiles({ moduleName, version, files }));
        });
    }

    closeFiles(filePaths: string[]) {
        this.store.dispatch(ModuleEditActions.closeFiles({ filePaths }));
    }

    // DEPENDENCIES
    changeAgentVersionDependency(version: string) {
        this.store.dispatch(ModuleEditActions.changeAgentVersion({ version }));
    }

    changeReceiveDataDependencies(dependencies: ModelsDependencyItem[]) {
        this.store.dispatch(ModuleEditActions.changeReceiveDataDependencies({ dependencies }));
    }

    changeSendDataDependencies(dependencies: ModelsDependencyItem[]) {
        this.store.dispatch(ModuleEditActions.changeSendDataDependencies({ dependencies }));
    }

    fetchAllModules() {
        this.store.dispatch(ModuleEditActions.fetchAllModules());
    }

    fetchAgentVersions() {
        this.store.dispatch(ModuleEditActions.fetchAgentVersions());
    }

    fetchModuleVersionsByName(moduleName: string) {
        this.store.dispatch(ModuleEditActions.fetchModuleVersionByName({ name: moduleName }));
    }

    // CHANGELOG
    deleteChangelogRecord(version: string) {
        this.store.dispatch(ModuleEditActions.deleteChangelogRecord({ version }));
    }

    updateChangelog(changelog: ModelsChangelog) {
        this.store.dispatch(ModuleEditActions.updateChangelogSection({ changelog }));
    }

    // STATE
    setValidationState(section: keyof ValidationState, status: boolean) {
        this.store.dispatch(ModuleEditActions.setValidationState({ section, status }));
    }

    resetOperationsError() {
        this.store.dispatch(ModuleEditActions.resetOperationErrors());
    }

    resetFile() {
        this.store.dispatch(ModuleEditActions.resetFile());
    }

    restoreState(): void {
        const params = this.activatedRoute.snapshot.queryParams as Record<string, string>;

        this.store.dispatch(ModuleEditActions.restoreState({ restoredState: {} }));
    }

    reset() {
        this.store.dispatch(ModuleEditActions.reset());
    }
}
