import { ChangeDetectionStrategy, Component, Inject, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslocoService } from '@ngneat/transloco';
import { PopUpPlacements, ThemePalette } from '@ptsecurity/mosaic/core';
import { McModalRef, McModalService } from '@ptsecurity/mosaic/modal';
import { McTabGroup } from '@ptsecurity/mosaic/tabs';
import {
    combineLatest,
    filter,
    forkJoin,
    from,
    map,
    Observable,
    reduce,
    shareReplay,
    Subscription,
    switchMap,
    take
} from 'rxjs';

import { ModelsModuleS, ModelsSemVersion, ModuleState } from '@soldr/api';
import { PERMISSIONS_TOKEN } from '@soldr/core';
import { CanLeavePage, LanguageService, ModuleVersionPipe, PageTitleService, ProxyPermission } from '@soldr/shared';
import { ModuleEditFacade } from '@soldr/store/modules';
import { SharedFacade } from '@soldr/store/shared';

import { ModuleSection } from '../../types';

enum TabsIndexes {
    General,
    Config,
    SecureConfig,
    Events,
    Actions,
    Fields,
    Dependencies,
    Localization,
    Files,
    Changelog
}

@Component({
    selector: 'soldr-edit-module-page',
    templateUrl: './edit-module-page.component.html',
    styleUrls: ['./edit-module-page.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class EditModulePageComponent implements OnInit, CanLeavePage, OnDestroy {
    @ViewChild('tabs') tabsEl: McTabGroup;

    @ViewChild('generalSection') generalSection: ModuleSection;
    @ViewChild('configSection') configSection: ModuleSection;
    @ViewChild('secureConfigSection') secureConfigSection: ModuleSection;
    @ViewChild('eventsSection') eventsSection: ModuleSection;
    @ViewChild('actionsSection') actionsSection: ModuleSection;
    @ViewChild('fieldsSection') fieldsSection: ModuleSection;
    @ViewChild('dependenciesSection') dependenciesSection: ModuleSection;
    @ViewChild('localizationSection') localizationSection: ModuleSection;
    @ViewChild('filesSection') filesSection: ModuleSection;
    @ViewChild('changelogSection') changelogSection: ModuleSection;

    availableVersions$ = this.moduleEditFacade.moduleVersions$.pipe(
        map((versions) => versions.map((item) => new ModuleVersionPipe().transform(item.info.version)))
    );
    canLeavePage = true;
    canRelease$ = this.moduleEditFacade.module$.pipe(map((module) => module?.state === ModuleState.Draft));
    canCreateDraft$ = combineLatest([this.moduleEditFacade.module$, this.moduleEditFacade.moduleVersions$]).pipe(
        map(
            ([module, versions]) =>
                module?.state === ModuleState.Release && versions.every((item) => item.state === ModuleState.Release)
        )
    );
    canUpdateModuleInPolicies$ = this.moduleEditFacade.canUpdateModuleInPolicies$;
    files$ = this.moduleEditFacade.files$;
    isDirty$ = this.moduleEditFacade.isDirty$;
    isLoading$ = this.moduleEditFacade.isLoadingModule$;
    isEmptyConfiguration$ = this.moduleEditFacade.module$.pipe(
        filter(Boolean),
        map(({ config_schema }) => Object.keys(config_schema.properties || {}).length === 0)
    );
    isEmptySecureConfiguration$ = this.moduleEditFacade.module$.pipe(
        filter(Boolean),
        map(({ secure_default_config }) => Object.keys(secure_default_config || {}).length === 0)
    );
    isEmptyEvents$ = this.moduleEditFacade.module$.pipe(
        filter(Boolean),
        map(
            ({ event_config_schema, default_event_config }) =>
                Object.keys(event_config_schema.properties || {}).length === 0 &&
                Object.keys(default_event_config || {}).length === 0
        )
    );
    isEmptyActions$ = this.moduleEditFacade.module$.pipe(
        filter(Boolean),
        map(
            ({ action_config_schema, default_action_config }) =>
                Object.keys(action_config_schema.properties || {}).length === 0 &&
                Object.keys(default_action_config || {}).length === 0
        )
    );
    isEmptyFields$ = this.moduleEditFacade.module$.pipe(
        filter(Boolean),
        map(({ fields_schema }) => Object.keys(fields_schema.properties || {}).length === 0)
    );
    isEmptyDependencies$ = this.moduleEditFacade.module$.pipe(
        filter(Boolean),
        map(({ static_dependencies }) => (static_dependencies || []).length === 0)
    );
    isSaving$ = this.moduleEditFacade.isSavingModule$;
    isUpdatingInPolicies$ = this.moduleEditFacade.isUpdatingModuleInPolicies$;
    isValidGeneral$ = this.moduleEditFacade.isValidGeneral$;
    isValidConfiguration$ = this.moduleEditFacade.isValidConfiguration$;
    isValidSecureConfiguration$ = this.moduleEditFacade.isValidSecureConfiguration$;
    isValidEvents$ = this.moduleEditFacade.isValidEvents$;
    isValidActions$ = this.moduleEditFacade.isValidActions$;
    isValidFields$ = this.moduleEditFacade.isValidFields$;
    isValidDependencies$ = this.moduleEditFacade.isValidDependencies$;
    isValidLocalization$ = this.moduleEditFacade.isValidLocalization$;
    isValidChangelog$ = this.moduleEditFacade.isValidChangelog$;
    language$ = this.languageService.current$;
    module$: Observable<ModelsModuleS> = this.moduleEditFacade.module$;
    ModuleState = ModuleState;
    popUpPlacements = PopUpPlacements;
    readOnly$ = this.moduleEditFacade.module$.pipe(map((module) => module?.state === ModuleState.Release));
    tabsMap = [
        'general',
        'configuration',
        'secure_configuration',
        'events',
        'actions',
        'fields',
        'dependencies',
        'localization',
        'files',
        'changelog'
    ];
    TabsIndexes = TabsIndexes;
    tabIndex = 0;
    themePalette = ThemePalette;
    versions$ = this.moduleEditFacade.moduleVersions$;

    private subscription = new Subscription();

    private onBeforeUnload = (event: BeforeUnloadEvent) => {
        if (!this.canLeavePage) {
            event.preventDefault();
            event.returnValue = '';

            return '';
        }

        return false;
    };

    constructor(
        private activatedRoute: ActivatedRoute,
        private languageService: LanguageService,
        private moduleEditFacade: ModuleEditFacade,
        private pageTitleService: PageTitleService,
        private router: Router,
        private sharedFacade: SharedFacade,
        private transloco: TranslocoService,
        private modalService: McModalService,
        @Inject(PERMISSIONS_TOKEN) public permitted: ProxyPermission
    ) {}

    get moduleName(): string {
        return this.activatedRoute.snapshot.params.name as string;
    }

    get version(): string {
        return (this.activatedRoute.snapshot.queryParams.version as string) || 'latest';
    }

    get tab(): string {
        return this.activatedRoute.snapshot.queryParams.tab as string;
    }

    ngOnInit(): void {
        this.defineTitle();
        this.restoreTabsState();

        this.loadModule(this.version);

        const dirtySubscription = this.moduleEditFacade.isDirty$.subscribe((v) => {
            this.canLeavePage = !v;
        });
        this.subscription.add(dirtySubscription);

        window.addEventListener('beforeunload', this.onBeforeUnload, { capture: true });
    }

    ngOnDestroy(): void {
        window.removeEventListener('beforeunload', this.onBeforeUnload);
    }

    onSelectTab() {
        this.saveState('tab', this.tabsEl?.tabs?.get(this.tabsEl.selectedIndex)?.tabId);
    }

    selectVersion(version: ModelsSemVersion) {
        const value = new ModuleVersionPipe().transform(version);

        this.saveState('version', value);
        this.loadModule(value);
    }

    updateInPolicies() {
        this.moduleEditFacade.updateModuleInPolicies(this.moduleName, this.version);
    }

    save() {
        combineLatest([this.moduleEditFacade.unusedFields$, this.moduleEditFacade.unusedRequiredFields$])
            .pipe(shareReplay({ bufferSize: 1, refCount: false }), take(1))
            .subscribe(([unusedFields, unusedRequiredFields]: [string[], string[]]) => {
                if (unusedFields?.length > 0) {
                    this.processUnusedFields(unusedFields);
                } else if (unusedRequiredFields?.length > 0) {
                    this.tabIndex = TabsIndexes.Fields;
                    this.modalService.create({
                        mcTitle: this.transloco.translate('modules.Modules.ModuleEdit.ModalTitle.FailedSaveModule'),
                        mcContent: this.transloco.translate('modules.Modules.ModuleEdit.Text.UnusedRequiredFields', {
                            fields: unusedRequiredFields
                        }),
                        mcCancelText: this.transloco.translate('common.Common.Pseudo.ButtonText.Close')
                    });
                } else {
                    this.doSave();
                }
            });
    }

    goToList() {
        this.router.navigate(['/modules']);
    }

    goLatestVersion() {
        this.versions$.pipe(shareReplay({ bufferSize: 1, refCount: false }), take(1)).subscribe((versions) => {
            const otherVersions = versions
                .map((item) => new ModuleVersionPipe().transform(item.info.version))
                .filter((version) => version !== this.version);

            if (otherVersions.length > 0) {
                this.loadModule('latest');
            } else {
                this.goToList();
            }
        });
    }

    private processUnusedFields(fields: string[]) {
        const removeUnusedFieldsModal: McModalRef = this.modalService.create({
            mcTitle: this.transloco.translate('modules.Modules.ModuleEdit.ModalTitle.FailedSaveModule'),
            mcContent: this.transloco.translate('modules.Modules.ModuleEdit.Text.RemoveUnusedFields', {
                fields: fields.join(', ')
            }),
            mcOkText: this.transloco.translate('common.Common.Pseudo.ButtonText.Delete'),
            mcCancelText: this.transloco.translate('common.Common.Pseudo.ButtonText.Cancel'),
            mcOnOk: () => removeUnusedFieldsModal.close(true),
            mcOnCancel: () => removeUnusedFieldsModal.close(false)
        });

        removeUnusedFieldsModal.afterClose.pipe(take(1)).subscribe((result) => {
            if (result) {
                for (const field of fields) {
                    this.moduleEditFacade.removeField(field);
                }

                this.doSave();
            }
        });
    }

    private doSave() {
        this.validateSections().subscribe((isOk) => {
            if (isOk) {
                this.moduleEditFacade.saveModule();
            }
        });
    }

    private validateSections() {
        return forkJoin([
            this.generalSection.validateForms(),
            this.configSection.validateForms(),
            this.secureConfigSection.validateForms(),
            this.eventsSection.validateForms(),
            this.actionsSection.validateForms(),
            this.fieldsSection.validateForms(),
            this.dependenciesSection.validateForms(),
            this.localizationSection.validateForms(),
            this.changelogSection.validateForms()
        ]).pipe(
            switchMap((statuses) => from(statuses)),
            reduce((acc, status) => acc && status, true)
        );
    }

    private defineTitle() {
        const moduleNameTitle = combineLatest([this.module$, this.language$]).pipe(
            filter(([group]) => Boolean(group)),
            switchMap(([module, lang]) =>
                this.transloco.selectTranslate(
                    'Modules.PageTitle.Text.Module',
                    { module: module.locale.module[lang].title },
                    'modules'
                )
            )
        );
        const titlesSubscription = combineLatest([
            moduleNameTitle,
            this.transloco.selectTranslate('Modules.PageTitle.Text.Modules', {}, 'modules'),
            this.sharedFacade.selectedServiceName$,
            this.transloco.selectTranslate('Shared.Pseudo.PageTitle.ApplicationName', {}, 'shared')
        ])
            .pipe(map((segments) => segments.filter(Boolean)))
            .subscribe((segments) => this.pageTitleService.setTitle(segments));

        this.subscription.add(titlesSubscription);
    }

    private loadModule(version: string) {
        this.moduleEditFacade.fetchModule(this.moduleName, version);
    }

    private restoreTabsState(): void {
        const tabIndex = this.tabsMap.indexOf(this.activatedRoute.snapshot.queryParams.tab as string);

        this.tabIndex = tabIndex === -1 ? 0 : tabIndex;
    }

    private saveState(key: string, value: string) {
        const queryParams = { [key]: value };

        this.router.navigate([], {
            relativeTo: this.activatedRoute,
            queryParams,
            queryParamsHandling: 'merge'
        });
    }
}
