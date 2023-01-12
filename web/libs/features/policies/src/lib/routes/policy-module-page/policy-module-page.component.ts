import { ChangeDetectionStrategy, Component, Inject, OnDestroy, OnInit, TemplateRef, ViewChild } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslocoService } from '@ngneat/transloco';
import { McSidepanelService } from '@ptsecurity/mosaic/sidepanel';
import { combineLatest, combineLatestWith, filter, map, pairwise, Subscription, switchMap, take } from 'rxjs';

import { ErrorResponse, ModuleStatus } from '@soldr/api';
import { PERMISSIONS_TOKEN } from '@soldr/core';
import {
    LanguageService,
    PageTitleService,
    STATE_STORAGE_TOKEN,
    StateStorage,
    ViewMode,
    mergeDeep,
    ModalInfoService,
    ProxyPermission
} from '@soldr/shared';
import { ModulesInstancesFacade } from '@soldr/store/modules-instances';
import { PoliciesFacade } from '@soldr/store/policies';
import { SharedFacade } from '@soldr/store/shared';

import { defaultPolicyModuleState, PolicyModuleState } from '../../utils';
import { PolicyModule } from '@soldr/models';

@Component({
    selector: 'soldr-policy-module-page',
    templateUrl: './policy-module-page.component.html',
    styleUrls: ['./policy-module-page.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class PolicyModulePageComponent implements OnInit, OnDestroy {
    policy$ = this.policiesFacade.policy$;
    isLoadingPolicy$ = this.policiesFacade.isLoadingPolicy$;
    isLoadingModules$ = this.policiesFacade.isLoadingModules$;

    isEnablingModule$ = this.modulesInstancesFacade.isEnablingModule$;
    isDeletingModule$ = this.modulesInstancesFacade.isDeletingModule$;
    isDisablingModule$ = this.modulesInstancesFacade.isDisablingModule$;
    isUpdatingModule$ = this.modulesInstancesFacade.isUpdatingModule$;
    isChangingModuleVersion$ = this.modulesInstancesFacade.isChangingVersionModule$;
    language$ = this.languageService.current$;

    module$ = this.policiesFacade.policyModules$.pipe(
        filter((policyModules: PolicyModule[]) => !!policyModules.length),
        map((policyModules: PolicyModule[]) =>
            policyModules.find(
                (policyModule: PolicyModule) =>
                    policyModule.info.name === this.activatedRoute.snapshot.params.moduleName
            )
        )
    );
    moduleEventsGridColumnFilterItems$ = this.policiesFacade.moduleEventsGridColumnFilterItems$;
    moduleVersions$ = this.modulesInstancesFacade.moduleVersions$;

    pageState: PolicyModuleState;
    viewModeEnum = ViewMode;
    subscription = new Subscription();
    moduleStatusEnum = ModuleStatus;

    @ViewChild('changeModuleVersionPanel') changeModuleVersionPanel: TemplateRef<any>;

    constructor(
        private activatedRoute: ActivatedRoute,
        private languageService: LanguageService,
        private modalInfoService: ModalInfoService,
        private modulesInstancesFacade: ModulesInstancesFacade,
        private pageTitleService: PageTitleService,
        private policiesFacade: PoliciesFacade,
        private router: Router,
        private sharedFacade: SharedFacade,
        private sidePanelService: McSidepanelService,
        private transloco: TranslocoService,
        @Inject(PERMISSIONS_TOKEN) public permitted: ProxyPermission,
        @Inject(STATE_STORAGE_TOKEN) private stateStorage: StateStorage
    ) {
        const { hash } = this.activatedRoute.snapshot.params as Record<string, string>;
        this.policiesFacade.fetchPolicy(hash);
    }

    ngOnInit(): void {
        this.defineTitle();

        this.pageState = mergeDeep(
            defaultPolicyModuleState(),
            (this.stateStorage.loadState('policyModule.view') as PolicyModuleState) || {}
        );

        const policySubscription = this.policy$.pipe(filter(Boolean)).subscribe((policy) => {
            const { moduleName } = this.activatedRoute.snapshot.params as Record<string, string>;

            this.modulesInstancesFacade.init(ViewMode.Policies, policy.id, moduleName);
            this.refreshModuleData();
        });
        this.subscription.add(policySubscription);
    }

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
    }

    doEnableModule(policyHash: string, moduleName: string) {
        this.isEnablingModule$
            .pipe(pairwise(), take(2), combineLatestWith(this.modulesInstancesFacade.enableError$))
            .subscribe(([[oldValue, newValue], enableError]) => {
                if (oldValue && !newValue) {
                    this.afterToggleModule(
                        enableError,
                        this.transloco.translate('shared.Shared.ModulesConfig.ErrorText.Enable')
                    );
                }
            });

        this.modulesInstancesFacade.enableModule(policyHash, moduleName);
    }

    doDisableModule(policyHash: string, moduleName: string) {
        this.isDisablingModule$
            .pipe(pairwise(), take(2), combineLatestWith(this.modulesInstancesFacade.disableError$))
            .subscribe(([[oldValue, newValue], disableError]) => {
                if (oldValue && !newValue) {
                    this.afterToggleModule(
                        disableError,
                        this.transloco.translate('shared.Shared.ModulesConfig.ErrorText.Disable')
                    );
                }
            });

        this.modulesInstancesFacade.disableModule(policyHash, moduleName);
    }

    doUpdateModule(version: string) {
        const { hash, moduleName } = this.activatedRoute.snapshot.params as Record<string, string>;
        this.isUpdatingModule$
            .pipe(pairwise(), take(2), combineLatestWith(this.modulesInstancesFacade.updateError$))
            .subscribe(([[oldValue, newValue], updateError]) => {
                if (oldValue && !newValue) {
                    if (!updateError) {
                        this.fetchPolicyModules();
                    } else {
                        this.modalInfoService.openErrorInfoModal(
                            this.transloco.translate('shared.Shared.ModulesConfig.ErrorText.Update')
                        );
                    }
                }
            });
        this.modulesInstancesFacade.updateModule(hash, moduleName, version);
    }

    doChangeVersion(version: string) {
        const { hash, moduleName } = this.activatedRoute.snapshot.params as Record<string, string>;
        this.isChangingModuleVersion$
            .pipe(pairwise(), take(2), combineLatestWith(this.modulesInstancesFacade.changeVersionError$))
            .subscribe(([[oldValue, newValue], changeVersionError]) => {
                if (oldValue && !newValue) {
                    if (!changeVersionError) {
                        this.fetchPolicyModules();
                    } else {
                        this.modalInfoService.openErrorInfoModal(
                            this.transloco.translate('shared.Shared.ModulesConfig.ErrorText.ChangeVersion')
                        );
                    }
                }
            });
        this.modulesInstancesFacade.changeModuleVersion(hash, moduleName, version);
    }

    openChangeVersionPanel() {
        const { moduleName } = this.activatedRoute.snapshot.params as Record<string, string>;
        this.sidePanelService.open(this.changeModuleVersionPanel);
        this.modulesInstancesFacade.fetchVersions(moduleName);
    }

    afterToggleModule(error: ErrorResponse, errorText: string) {
        if (!error) {
            this.refreshModuleData();
        } else {
            this.modalInfoService.openErrorInfoModal(errorText);
        }
    }

    private refreshModuleData() {
        this.fetchPolicyModules();
        this.modulesInstancesFacade.fetchEvents();
        this.modulesInstancesFacade.fetchModuleEventsFilterItems();
        this.sharedFacade.fetchAllGroups();
    }

    private defineTitle() {
        const moduleNameTitle = combineLatest([this.module$, this.language$]).pipe(
            filter(([module]) => Boolean(module)),
            switchMap(([module, lang]) =>
                this.transloco.selectTranslate(
                    'Policies.PageTitle.Text.Module',
                    { module: module.locale.module[lang].title },
                    'policies'
                )
            )
        );
        const policyNameTitle = combineLatest([this.policy$, this.language$]).pipe(
            filter(([policy]) => Boolean(policy)),
            switchMap(([policy, lang]) =>
                this.transloco.selectTranslate(
                    'Policies.PageTitle.Text.Policy',
                    { policy: policy.info.name[lang] },
                    'policies'
                )
            )
        );

        const titlesSubscription = combineLatest([
            moduleNameTitle,
            this.transloco.selectTranslate('Policies.PageTitle.Text.Modules', {}, 'policies'),
            policyNameTitle,
            this.transloco.selectTranslate('Policies.PageTitle.Text.Policies', {}, 'policies'),
            this.sharedFacade.selectedServiceName$,
            this.transloco.selectTranslate('Shared.Pseudo.PageTitle.ApplicationName', {}, 'shared')
        ])
            .pipe(map((segments) => segments.filter(Boolean)))
            .subscribe((segments) => this.pageTitleService.setTitle(segments));

        this.subscription.add(titlesSubscription);
    }

    fetchPolicyModules() {
        const { hash } = this.activatedRoute.snapshot.params as Record<string, string>;

        this.policiesFacade.fetchModules(hash);
    }
}
