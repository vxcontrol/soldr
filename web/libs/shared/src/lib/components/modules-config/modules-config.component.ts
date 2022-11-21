import {
    Component,
    EventEmitter,
    forwardRef,
    Inject,
    Input,
    OnChanges,
    OnDestroy,
    OnInit,
    Output,
    SimpleChanges,
    TemplateRef,
    ViewChild
} from '@angular/core';
import { Router } from '@angular/router';
import { TranslocoService } from '@ngneat/transloco';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { McListSelectionChange } from '@ptsecurity/mosaic/list';
import { McSidepanelService } from '@ptsecurity/mosaic/sidepanel';
import {
    BehaviorSubject,
    combineLatest,
    combineLatestWith,
    filter,
    map,
    Observable,
    pairwise,
    ReplaySubject,
    Subscription,
    take,
    withLatestFrom
} from 'rxjs';

import { DependencyType, ErrorResponse, ModelsModuleA, ModuleStatus } from '@soldr/api';
import { PERMISSIONS_TOKEN } from '@soldr/core';
import { Policy, PolicyModule } from '@soldr/models';
import { ModalInfoService, ModuleConfigBlockComponent, ProxyPermission, sortModules } from '@soldr/shared';
import { ModulesInstancesFacade } from '@soldr/store/modules-instances';
import { SharedFacade } from '@soldr/store/shared';

import { LanguageService } from '../../services';
import { EntityDependency, EntityModule, ViewMode } from '../../types';

@Component({
    selector: 'soldr-modules-config',
    templateUrl: './modules-config.component.html',
    styleUrls: ['./modules-config.component.scss']
})
export class ModulesConfigComponent implements OnInit, OnChanges, OnDestroy {
    @Input() dependencies: EntityDependency[];
    @Input() moduleInstanceLinkTemplate: TemplateRef<any>;
    @Input() modules: EntityModule[];
    @Input() policy: Policy;
    @Input() selectModuleName: string;
    @Input() viewMode: ViewMode;

    @Output() selectModule = new EventEmitter<EntityModule>();
    @Output() refresh = new EventEmitter();
    @Output() afterChangeModuleState = new EventEmitter();

    @ViewChild('changeModuleVersionPanel') changeModuleVersionPanel: TemplateRef<any>;
    @ViewChild(forwardRef(() => ModuleConfigBlockComponent)) moduleConfig: ModuleConfigBlockComponent;

    canShowUpgradeModule = true;
    consistencyByModuleName$: Observable<Record<string, boolean>>;
    dependencies$ = new BehaviorSubject<EntityDependency[]>([]);
    disabledModules$: Observable<EntityModule[]>;
    enabledModules$: Observable<EntityModule[]>;
    eventsSearch$ = new BehaviorSubject('');
    isChangingModuleVersion$ = this.modulesInstancesFacade.isChangingVersionModule$;
    isDeletingModule$ = this.modulesInstancesFacade.isDeletingModule$;
    isDisablingModule$ = this.modulesInstancesFacade.isDisablingModule$;
    isEnablingModule$ = this.modulesInstancesFacade.isEnablingModule$;
    isLoading$: Observable<boolean>;
    isSavingModule$ = this.modulesInstancesFacade.isSavingModule$;
    isReadOnly = false;
    language$ = this.languageService.current$;
    moduleStatusEnum = ModuleStatus;
    modules$ = new BehaviorSubject<EntityModule[]>([]);
    notInstalledModules$: Observable<EntityModule[]>;
    selected: string[] = [];
    selectedModule$: Observable<EntityModule>;
    selectedModuleName$ = new ReplaySubject<string>(1);
    selectedModuleVersions$ = this.modulesInstancesFacade.moduleVersions$;
    sortModules = sortModules;
    subscription = new Subscription();
    themePalette = ThemePalette;
    unavailableForInstallation$: Observable<EntityModule[]>;
    viewModeEnum = ViewMode;

    constructor(
        private languageService: LanguageService,
        private modalInfoService: ModalInfoService,
        private modulesInstancesFacade: ModulesInstancesFacade,
        private router: Router,
        private sharedFacade: SharedFacade,
        private sidePanelService: McSidepanelService,
        private transloco: TranslocoService,
        @Inject(PERMISSIONS_TOKEN) public permitted: ProxyPermission
    ) {}

    ngOnInit(): void {
        this.defineObservables();
    }

    ngOnChanges({ modules, dependencies, selectModuleName }: SimpleChanges): void {
        if (modules?.currentValue) {
            this.modules$.next(this.modules);
        }

        if (dependencies?.currentValue) {
            this.dependencies$.next(this.dependencies);
        }

        if (selectModuleName?.currentValue) {
            this.initSelection(this.selectModuleName);
        }
    }

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
    }

    onChangeSelectedModule($event: McListSelectionChange) {
        if (!$event.option.selected) {
            return;
        }

        const selectedModuleName = $event.option.value as string;
        if (selectedModuleName) {
            this.selectedModuleName$.next(selectedModuleName);
        }
    }

    doEnableModule(module: EntityModule) {
        this.modulesInstancesFacade.enableModule(this.policy.hash, module.info.name);
    }

    doDisableModule(module: EntityModule) {
        this.modulesInstancesFacade.disableModule(this.policy.hash, module.info.name);
    }

    doOpenChangeModuleVersionPanel(module: EntityModule) {
        const moduleName = module.info.name;
        this.sidePanelService.open(this.changeModuleVersionPanel);
        this.modulesInstancesFacade.fetchVersions(moduleName);
    }

    doChangeVersion(moduleName: string, version: string) {
        this.modulesInstancesFacade.changeModuleVersion(this.policy?.hash, moduleName, version);
    }

    afterToggleModule(error: ErrorResponse, errorText: string) {
        if (!error) {
            this.refresh.emit();
            this.afterChangeModuleState.emit();
        } else {
            this.modalInfoService.openErrorInfoModal(errorText);
        }
    }

    save(module: ModelsModuleA) {
        this.moduleConfig.validate().then(({ result }) => {
            if (result) {
                const model = this.moduleConfig.getModel();
                const updatedModule = { ...module, current_config: model };

                this.saveModuleEventConfig(updatedModule);
            }
        });
    }

    cancel() {
        this.moduleConfig.reset();
    }

    saveModuleEventConfig(updatedModule: EntityModule) {
        this.isSavingModule$
            .pipe(pairwise(), take(2), withLatestFrom(this.modules$))
            .subscribe(([[oldValue, newValue], modules]: [[boolean, boolean], EntityModule[]]) => {
                if (oldValue && !newValue) {
                    this.afterChangeModuleState.emit();
                }
            });

        this.modulesInstancesFacade.saveModuleConfig(this.policy.hash, updatedModule);
    }

    get isDirtyConfig() {
        return this.moduleConfig?.isDirty;
    }

    private defineObservables() {
        this.defineListsObservables();
        this.defineSelectionObservables();
        this.defineOperationsObservables();
    }

    private defineOperationsObservables() {
        const disablingSubscription = this.isDisablingModule$
            .pipe(pairwise(), combineLatestWith(this.modulesInstancesFacade.disableError$))
            .subscribe(([[oldValue, newValue], disableError]) => {
                if (oldValue && !newValue) {
                    this.afterToggleModule(
                        disableError,
                        this.transloco.translate('shared.Shared.ModulesConfig.ErrorText.Disable')
                    );
                }
            });
        this.subscription.add(disablingSubscription);

        const enablingSubscription = this.isEnablingModule$
            .pipe(pairwise(), combineLatestWith(this.modulesInstancesFacade.enableError$))
            .subscribe(([[oldValue, newValue], enableError]) => {
                if (oldValue && !newValue) {
                    this.afterToggleModule(
                        enableError,
                        this.transloco.translate('shared.Shared.ModulesConfig.ErrorText.Enable')
                    );
                }
            });
        this.subscription.add(enablingSubscription);

        const deletingSubscription = this.isDeletingModule$
            .pipe(pairwise(), combineLatestWith(this.modulesInstancesFacade.deleteError$))
            .subscribe(([[oldValue, newValue], deleteError]) => {
                if (oldValue && !newValue && !deleteError) {
                    this.refresh.emit();
                }
            });
        this.subscription.add(deletingSubscription);

        const changingVersionSubscription = this.isChangingModuleVersion$
            .pipe(pairwise(), combineLatestWith(this.modulesInstancesFacade.changeVersionError$))
            .subscribe(([[oldValue, newValue], changeVersionError]) => {
                if (oldValue && !newValue) {
                    if (!changeVersionError) {
                        this.refresh.emit();
                    } else {
                        this.modalInfoService.openErrorInfoModal(
                            this.transloco.translate('shared.Shared.ModulesConfig.ErrorText.ChangeVersion')
                        );
                    }
                }
            });
        this.subscription.add(changingVersionSubscription);
    }

    private defineSelectionObservables() {
        this.selectedModule$ = combineLatest([this.modules$, this.selectedModuleName$]).pipe(
            map(([modules, name]) => modules.find((module) => module.info.name === name))
        );

        const defaultSelection = combineLatest([
            this.enabledModules$,
            this.disabledModules$,
            this.notInstalledModules$,
            this.unavailableForInstallation$
        ]).subscribe(([enabledModules, disabledModules, notInstalledModules, unavailable]) => {
            if (!this.selectModuleName) {
                const defaultModuleName = [
                    ...enabledModules,
                    ...disabledModules,
                    ...notInstalledModules,
                    ...unavailable
                ][0]?.info.name;
                this.initSelection(defaultModuleName);
            }
        });
        this.subscription.add(defaultSelection);

        const selectModuleSubscription = this.selectedModule$.subscribe((module: EntityModule) => {
            if (module && this.selectModuleName !== module.info.name) {
                setTimeout(() => {
                    this.modulesInstancesFacade.resetModuleErrors();
                    this.selectModule.emit(module);
                });
            }
        });
        this.subscription.add(selectModuleSubscription);

        const selectModule = this.selectedModule$.pipe(filter(Boolean)).subscribe(() => {
            this.modulesInstancesFacade.fetchVersions(this.selectModuleName);
            this.canShowUpgradeModule = true;
            this.eventsSearch$.next('');
            this.isReadOnly =
                this.viewMode !== ViewMode.Policies ||
                (this.viewMode === ViewMode.Policies && this.policy?.info?.system) ||
                !this.permitted.EditPolicies;
        });
        this.subscription.add(selectModule);
    }

    private defineListsObservables() {
        this.consistencyByModuleName$ = this.dependencies$.pipe(
            map((dependencies) =>
                dependencies
                    .filter((dependency) =>
                        [DependencyType.ToMakeAction, DependencyType.ToReceiveData, DependencyType.ToSendData].includes(
                            dependency.type
                        )
                    )
                    .reduce((acc, dependency) => {
                        const prevValue = acc[dependency.source_module_name];

                        return {
                            ...acc,
                            [dependency.source_module_name]:
                                prevValue === undefined ? dependency.status : prevValue && dependency.status
                        };
                    }, {} as Record<string, boolean>)
            )
        );

        this.enabledModules$ = combineLatest([this.modules$, this.consistencyByModuleName$, this.language$]).pipe(
            map(([modules, consistencyByModuleName, language]) =>
                modules
                    .filter((item) => item.status === ModuleStatus.Joined)
                    .sort((a, b) => {
                        const consistencyA =
                            consistencyByModuleName[a.info.name] || consistencyByModuleName[a.info.name] === undefined;
                        const consistencyB =
                            consistencyByModuleName[b.info.name] || consistencyByModuleName[b.info.name] === undefined;

                        if (consistencyA && !consistencyB) {
                            return 1;
                        } else if (!consistencyA && consistencyB) {
                            return -1;
                        }

                        return sortModules(language)(a, b);
                    })
            )
        );

        this.disabledModules$ = combineLatest([this.modules$, this.language$]).pipe(
            map(([modules, language]) =>
                modules
                    .filter((item: PolicyModule) => item.status === ModuleStatus.Inactive && item.details?.exists)
                    .sort((a, b) => sortModules(language)(a, b))
            )
        );

        this.notInstalledModules$ = combineLatest([this.modules$, this.language$]).pipe(
            map(([modules, language]) =>
                modules
                    .filter(
                        (item: PolicyModule) =>
                            item?.details.exists === false &&
                            item?.details.duplicate === false &&
                            this.policy?.info?.system === false
                    )
                    .sort((a, b) => sortModules(language)(a, b))
            )
        );

        this.unavailableForInstallation$ = combineLatest([this.modules$, this.language$]).pipe(
            map(([modules, language]) =>
                modules
                    .filter(
                        (item: PolicyModule) =>
                            (item?.details.duplicate || this.policy?.info?.system) && item?.details.exists === false
                    )
                    .sort((a, b) => sortModules(language)(a, b))
            )
        );
    }

    private initSelection(moduleName: string) {
        this.selected = [moduleName];
        this.selectedModuleName$.next(moduleName);
    }
}
