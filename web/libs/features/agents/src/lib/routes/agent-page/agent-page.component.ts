import {
    AfterViewInit,
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    Inject,
    OnDestroy,
    OnInit,
    ViewChild
} from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslocoService } from '@ngneat/transloco';
import { SidebarPositions } from '@ptsecurity/mosaic/sidebar';
import { Direction } from '@ptsecurity/mosaic/splitter';
import { McTabGroup } from '@ptsecurity/mosaic/tabs';
import {
    combineLatest,
    combineLatestWith,
    filter,
    map,
    Observable,
    pairwise,
    Subscription,
    switchMap,
    take
} from 'rxjs';

import { PERMISSIONS_TOKEN } from '@soldr/core';
import { AgentPageState, defaultAgentPageState } from '@soldr/features/agents';
import { Agent, AgentUpgradeTask, canUpgradeAgent } from '@soldr/models';
import {
    EntityModule,
    Filtration,
    DateFormatterService,
    LanguageService,
    PageTitleService,
    Sorting,
    STATE_STORAGE_TOKEN,
    StateStorage,
    ViewMode,
    WidthChangeEvent,
    mergeDeep,
    ProxyPermission
} from '@soldr/shared';
import { AgentCardFacade } from '@soldr/store/agents';
import { SharedFacade } from '@soldr/store/shared';
import { TagDomain, TagsFacade } from '@soldr/store/tags';

import { AgentDependenciesFacadeService } from '../../services';

@Component({
    selector: 'soldr-agent-page',
    templateUrl: './agent-page.component.html',
    styleUrls: ['./agent-page.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class AgentPageComponent implements OnInit, AfterViewInit, OnDestroy {
    agent$: Observable<Agent> = this.agentCardFacade.agent$;
    allGroups$ = this.sharedFacade.allGroups$;
    canBlockAgent$: Observable<boolean>;
    canUpgradeAgent$: Observable<boolean>;
    deleteError$ = this.agentCardFacade.deleteError$;
    dependencies$ = this.agentDependenciesFacadeService.$dependencies$;
    direction = Direction;
    events$ = this.agentCardFacade.agentEvents$.pipe(
        map((events) => this.dateFormatterService.formatToAbsoluteLongWithSeconds(events, 'date'))
    );
    eventGridColumnFilterItems$ = this.agentCardFacade.eventGridColumnFilterItems$;
    eventsGridFiltrationByFields$ = this.agentCardFacade.eventsGridFiltrationByFields$;
    eventsPage$ = this.agentCardFacade.eventsPage$;
    eventsSearchValue$ = this.agentCardFacade.eventsSearchValue$;
    isBlockingAgent$ = this.agentCardFacade.isBlockingAgent$;
    isCancelUpgradingAgent$ = this.agentCardFacade.isCancelUpgradingAgent$;
    isDeletingAgent$ = this.agentCardFacade.isDeletingAgent$;
    isDeletingFromGroup$ = this.agentCardFacade.isDeletingFromGroup$;
    isLoadingAgent$ = this.agentCardFacade.isLoadingAgent$;
    isLoadingAllGroups$ = this.sharedFacade.isLoadingAllGroups$;
    isLoadingEvents$ = this.agentCardFacade.isLoadingEvents$;
    isLoadingLatestBinary$ = this.sharedFacade.isLoadingLatestAgentBinary$;
    isLoadingModules$ = this.sharedFacade.isLoadingAllModules$;
    isMovingAgent$ = this.agentCardFacade.isMovingAgent$;
    isUpdatingAgent$ = this.agentCardFacade.isUpdatingAgent$;
    isUpgradingAgent$ = this.agentCardFacade.isUpgradingAgent$;
    language$ = this.languageService.current$;
    latestAgentBinaryVersion$ = this.sharedFacade.latestAgentBinaryVersion$;
    modules$ = this.agentCardFacade.agentModules$;
    moveToGroupError$ = this.agentCardFacade.moveToGroupError$;
    pageState: AgentPageState;
    selectedModuleName: string;
    sidebarPositions = SidebarPositions;
    subscription = new Subscription();
    tabIndex = 0;
    totalEvents$ = this.agentCardFacade.totalEvents$;
    updateError$ = this.agentCardFacade.updateError$;
    viewModeEnum = ViewMode;

    @ViewChild('tabs') tabsEl: McTabGroup;

    constructor(
        private activatedRoute: ActivatedRoute,
        private agentCardFacade: AgentCardFacade,
        private agentDependenciesFacadeService: AgentDependenciesFacadeService,
        private changeDetectorRef: ChangeDetectorRef,
        private dateFormatterService: DateFormatterService,
        private languageService: LanguageService,
        private pageTitleService: PageTitleService,
        private router: Router,
        private sharedFacade: SharedFacade,
        private tagsFacade: TagsFacade,
        private transloco: TranslocoService,
        @Inject(PERMISSIONS_TOKEN) public permitted: ProxyPermission,
        @Inject(STATE_STORAGE_TOKEN) private stateStorage: StateStorage
    ) {
        this.refreshData();
    }

    ngOnInit(): void {
        this.defineTitle();

        this.pageState = mergeDeep(
            defaultAgentPageState(),
            (this.stateStorage.loadState('agent.view') as AgentPageState) || {}
        );

        this.canUpgradeAgent$ = combineLatest([
            this.agentCardFacade.agent$,
            this.sharedFacade.latestAgentBinaryVersion$
        ]).pipe(map(([agent, latestBinaryVersion]: [Agent, string]) => canUpgradeAgent(agent, latestBinaryVersion)));

        this.canBlockAgent$ = this.agentCardFacade.agent$.pipe(map((agent) => agent?.auth_status !== 'blocked'));

        const blockingSubscription: Subscription = this.agentCardFacade.isBlockingAgent$
            .pipe(pairwise())
            .subscribe(([oldValue, newValue]) => {
                if (oldValue && !newValue) {
                    this.refreshData();
                }
            });
        this.subscription.add(blockingSubscription);

        const isMovingAgentsSubscription: Subscription = this.isMovingAgent$
            .pipe(pairwise(), combineLatestWith(this.moveToGroupError$))
            .subscribe(([[oldValue, newValue], moveToGroupError]) => {
                if (oldValue && !newValue && !moveToGroupError) {
                    this.sharedFacade.fetchAllGroups();
                }
            });
        this.subscription.add(isMovingAgentsSubscription);

        this.selectedModuleName = this.activatedRoute.snapshot.queryParams.moduleName as string;
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
        this.agentCardFacade.reset();
        this.subscription.unsubscribe();
    }

    saveLeftSidebarState(opened: boolean) {
        this.pageState.leftSidebar = { ...this.pageState.leftSidebar, opened };
    }

    saveLeftSidebarWidth($event: WidthChangeEvent) {
        if ($event.width !== '32px') {
            this.pageState.leftSidebar = { ...this.pageState.leftSidebar, width: $event.width };
        }
    }

    selectTag(tag: string) {
        this.router.navigate(['/agents'], {
            queryParams: {
                filtration: JSON.stringify([
                    {
                        field: 'tags',
                        value: [tag]
                    }
                ])
            }
        });
    }

    refreshData() {
        const { hash } = this.activatedRoute.snapshot.params;
        this.agentCardFacade.fetchAgent(hash as string);
        this.sharedFacade.fetchAllModules();
        this.sharedFacade.fetchLatestAgentBinary();
        this.tagsFacade.fetchTags(TagDomain.Agents);
        this.agentCardFacade.agent$.pipe(filter(Boolean), take(1)).subscribe(({ id }) => {
            this.agentCardFacade.fetchEvents(id);
            this.agentCardFacade.fetchEventFilterItems();
        });
    }

    refreshGroups() {
        this.sharedFacade.fetchAllGroups();
    }

    upgradeAgent(agent: Agent, latestAgentBinaryVersion: string) {
        this.agentCardFacade.upgradeAgent(agent, latestAgentBinaryVersion);
    }

    blockAgent(agent: Agent) {
        this.agentCardFacade.blockAgent(agent.id);
    }

    onSelectTab() {
        this.saveState();
    }

    loadNextEventsPage(id: number, page: number) {
        this.agentCardFacade.fetchEvents(id, page);
    }

    eventsSearch(value: string) {
        this.agentCardFacade.setEventsGridSearch(value);
    }

    eventsFilter(value: Filtration) {
        this.agentCardFacade.setEventsGridFiltration(value);
    }

    eventsSort(sorting: Sorting) {
        this.agentCardFacade.setEventsGridSorting(sorting);
    }

    onAfterDelete() {
        this.router.navigate(['/agents']);
    }

    resetFiltration() {
        this.agentCardFacade.resetEventsFiltration();
    }

    onSelectModule(module: EntityModule) {
        this.selectedModuleName = module?.info.name;
        this.saveState();
    }

    onRefresh() {
        this.refreshData();
        this.changeDetectorRef.detectChanges();
    }

    moveToGroup(event: { ids: number[]; groupId: number }) {
        this.agentCardFacade.moveToGroup(event.ids[0], event.groupId);
    }

    moveToNewGroup(event: { ids: number[]; groupName: string }) {
        this.agentCardFacade.moveToNewGroup(event.ids[0], event.groupName);
    }

    delete([id]: [number]) {
        this.agentCardFacade.deleteAgent(id);
    }

    update(event: Agent) {
        this.agentCardFacade.updateAgent(event);
    }

    upgrade(event: { agents: Agent[]; version: string }) {
        this.agentCardFacade.upgradeAgent(event.agents[0], event.version);
    }

    cancelUpgrade(event: { hash: string; task: AgentUpgradeTask }) {
        this.agentCardFacade.cancelUpgradeAgent(event.hash, event.task);
    }

    private saveState() {
        const tab = this.tabsEl.tabs.get(this.tabsEl.selectedIndex)?.tabId;
        const queryParams: Record<string, string> = {
            tab,
            moduleName: tab === 'modules' ? this.selectedModuleName : undefined
        };

        this.router.navigate([], {
            relativeTo: this.activatedRoute,
            queryParams,
            queryParamsHandling: 'merge',
            replaceUrl: true
        });
    }

    private defineTitle() {
        const agentNameTitle = this.agent$.pipe(
            filter(Boolean),
            switchMap((agent) =>
                this.transloco.selectTranslate('Agents.PageTitle.Text.Agent', { agent: agent.description }, 'agents')
            )
        );
        const titlesSubscription = combineLatest([
            agentNameTitle,
            this.transloco.selectTranslate('Agents.PageTitle.Text.Agents', {}, 'agents'),
            this.sharedFacade.selectedServiceName$,
            this.transloco.selectTranslate('Shared.Pseudo.PageTitle.ApplicationName', {}, 'shared')
        ])
            .pipe(map((segments) => segments.filter(Boolean)))
            .subscribe((segments) => this.pageTitleService.setTitle(segments));

        this.subscription.add(titlesSubscription);
    }
}
