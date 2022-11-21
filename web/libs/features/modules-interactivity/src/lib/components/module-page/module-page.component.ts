import {
    AfterViewInit,
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
    ViewChild
} from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { SidebarPositions } from '@ptsecurity/mosaic/sidebar';
import { Direction } from '@ptsecurity/mosaic/splitter';
import { McTabGroup } from '@ptsecurity/mosaic/tabs';
import { BehaviorSubject, combineLatestWith, filter, map, Observable, pairwise, Subscription, take } from 'rxjs';

import { ErrorResponse, ModelsModuleA } from '@soldr/api';
import { PERMISSIONS_TOKEN } from '@soldr/core';
import { AgentModuleState, AgentPageState } from '@soldr/features/agents';
import { Policy } from '@soldr/models';
import {
    DateFormatterService,
    LanguageService,
    GridColumnFilterItem,
    Sorting,
    ModuleConfigBlockComponent,
    STATE_STORAGE_TOKEN,
    StateStorage,
    WidthChangeEvent,
    Entity,
    EntityModule,
    Filtration,
    ProxyPermission,
    ViewMode
} from '@soldr/shared';
import { ModulesInstancesFacade } from '@soldr/store/modules-instances';
import { SharedFacade } from '@soldr/store/shared';

@Component({
    selector: 'soldr-module-page',
    templateUrl: './module-page.component.html',
    styleUrls: ['./module-page.component.scss']
})
export class ModulePageComponent implements OnInit, OnChanges, AfterViewInit, OnDestroy {
    @Input() entity: Entity;
    @Input() eventsGridFilter: { [field: string]: GridColumnFilterItem[] };
    @Input() hasModuleOperations = false;
    @Input() state: AgentModuleState;
    @Input() stateStorageKey: string;
    @Input() viewMode: ViewMode;

    @Output() update = new EventEmitter();
    @Output() seeVersions = new EventEmitter();

    @ViewChild('tabs') tabsEl: McTabGroup;
    @ViewChild(forwardRef(() => ModuleConfigBlockComponent)) moduleConfig: ModuleConfigBlockComponent;

    direction = Direction;
    events$ = this.modulesInstancesFacade.events$.pipe(
        map((events) => this.dateFormatterService.formatToAbsoluteLongWithSeconds(events, 'date'))
    );
    eventsGridFiltration$ = this.modulesInstancesFacade.eventsGridFiltration$;
    eventsGridFiltrationByFields$ = this.modulesInstancesFacade.eventsGridFiltrationByFields$;
    eventsPage$ = this.modulesInstancesFacade.eventsPage$;
    eventsSearchValue$ = this.modulesInstancesFacade.eventsSearchValue$;
    isLoadedModuleManagement$ = new BehaviorSubject<boolean>(false);
    isLoadingEvents$ = this.modulesInstancesFacade.isLoadingEvents$;
    isLoadingModule$ = this.modulesInstancesFacade.isLoadingModule$;
    isLoadingModules$ = this.sharedFacade.isLoadingAllModules$;
    isSavingModule$ = this.modulesInstancesFacade.isSavingModule$;
    isReadOnly = false;
    language$ = this.languageService.current$;
    module$: Observable<ModelsModuleA> = this.modulesInstancesFacade.module$;
    moduleVersions$ = this.modulesInstancesFacade.moduleVersions$;
    modules$ = this.sharedFacade.allModules$;
    pageState: AgentPageState;
    saveError$: Observable<ErrorResponse> = this.modulesInstancesFacade.saveError$;
    sidebarPositions = SidebarPositions;
    subscription = new Subscription();
    tabIndex = 0;
    themePalette = ThemePalette;
    totalEvents$ = this.modulesInstancesFacade.totalEvents$;
    viewModeEnum = ViewMode;

    constructor(
        private dateFormatterService: DateFormatterService,
        private modulesInstancesFacade: ModulesInstancesFacade,
        private activatedRoute: ActivatedRoute,
        private languageService: LanguageService,
        private sharedFacade: SharedFacade,
        private route: ActivatedRoute,
        private router: Router,
        @Inject(STATE_STORAGE_TOKEN) private stateStorage: StateStorage,
        @Inject(PERMISSIONS_TOKEN) public permitted: ProxyPermission
    ) {}

    ngOnInit(): void {
        this.defineObservables();

        this.isReadOnly =
            this.viewMode !== ViewMode.Policies ||
            (this.viewMode === ViewMode.Policies && (this.entity as Policy)?.info?.system) ||
            !this.permitted.EditPolicies;
    }

    ngOnChanges({ hasModuleOperations }: SimpleChanges): void {
        if (hasModuleOperations?.previousValue && !hasModuleOperations?.currentValue) {
            this.tabIndex = 0;
        }
    }

    ngAfterViewInit(): void {
        const tabIndex = this.tabsEl.tabs
            .toArray()
            .findIndex((tab) => tab.tabId === this.activatedRoute.snapshot.queryParams.tab);

        setTimeout(() => {
            this.tabIndex = tabIndex === -1 ? 0 : tabIndex;
        });
    }

    ngOnDestroy(): void {
        this.modulesInstancesFacade.reset();
        this.subscription.unsubscribe();
    }

    saveModuleConfig(module: ModelsModuleA) {
        this.moduleConfig.validate().then(({ result }) => {
            if (result) {
                const model = this.moduleConfig.getModel();
                const updatedModule = { ...module, current_config: model };

                this.saveModuleEventConfig(updatedModule);
            }
        });
    }

    saveModuleEventConfig(updatedModule: EntityModule) {
        this.isSavingModule$
            .pipe(pairwise(), take(2), combineLatestWith(this.saveError$))
            .subscribe(([[oldValue, newValue], saveError]: [[boolean, boolean], ErrorResponse]) => {
                if (oldValue && !newValue && !saveError) {
                    this.afterSaveConfig();
                }
            });

        this.modulesInstancesFacade.saveModuleConfig(this.entity.hash, updatedModule);
    }

    saveLeftSidebarState(opened: boolean) {
        this.state.leftSidebar = { ...this.state.leftSidebar, opened };
    }

    saveLeftSidebarWidth($event: WidthChangeEvent) {
        if ($event.width !== '32px') {
            this.state.leftSidebar = { ...this.state.leftSidebar, width: $event.width };
        }
    }

    onSelectTab() {
        this.saveState();
    }

    eventsSearch(value: string) {
        this.modulesInstancesFacade.setEventsGridSearch(value);
    }

    eventsFilter(value: Filtration) {
        this.modulesInstancesFacade.setEventsGridFiltration(value);
    }

    loadNextEventsPage(page: number) {
        this.modulesInstancesFacade.fetchEvents(page);
    }

    doUpdate(version: string) {
        this.update.emit(version);
    }

    doSeeVersions() {
        this.seeVersions.emit();
    }

    eventsResetFiltration() {
        this.modulesInstancesFacade.resetEventsFiltration();
    }

    eventsSort(sorting: Sorting) {
        this.modulesInstancesFacade.setEventsGridSorting(sorting);
    }

    get hasManagementTab() {
        return this.viewMode === ViewMode.Agents && this.permitted.ViewModulesOperations && this.hasModuleOperations;
    }

    private defineObservables() {
        const moduleSubscription = this.module$.pipe(filter(Boolean)).subscribe(() => {
            this.refreshData();
        });
        this.subscription.add(moduleSubscription);
    }

    private saveState() {
        const queryParams = {
            tab: this.tabsEl.tabs.get(this.tabsEl.selectedIndex)?.tabId
        };

        this.router.navigate([], {
            relativeTo: this.activatedRoute,
            queryParams,
            queryParamsHandling: 'merge',
            replaceUrl: true
        });
    }

    private afterSaveConfig() {
        this.refreshData();
        this.modulesInstancesFacade.fetchEvents();
        this.modulesInstancesFacade.fetchModule(this.entity.hash);
    }

    private refreshData() {
        this.sharedFacade.fetchAllModules();
        this.modulesInstancesFacade.fetchVersions(this.activatedRoute.snapshot.params.moduleName as string);
    }
}
