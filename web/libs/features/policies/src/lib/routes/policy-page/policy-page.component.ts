import { AfterViewInit, ChangeDetectionStrategy, Component, Inject, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslocoService } from '@ngneat/transloco';
import { SidebarPositions } from '@ptsecurity/mosaic/sidebar';
import { Direction } from '@ptsecurity/mosaic/splitter';
import { McTabGroup } from '@ptsecurity/mosaic/tabs';
import { combineLatest, filter, map, merge, pairwise, Subscription, switchMap } from 'rxjs';

import { ModelsModuleA } from '@soldr/api';
import { PERMISSIONS_TOKEN } from '@soldr/core';
import { Agent, AgentUpgradeTask, Group, Policy } from '@soldr/models';
import {
    DateFormatterService,
    EntityModule,
    Filtration,
    LanguageService,
    LinkPolicyToGroupFacade,
    mergeDeep,
    PageTitleService,
    POLICY_LINKING_FACADE,
    ProxyPermission,
    Sorting,
    STATE_STORAGE_TOKEN,
    StateStorage,
    ViewMode,
    WidthChangeEvent
} from '@soldr/shared';
import { PoliciesFacade } from '@soldr/store/policies';
import { SharedFacade } from '@soldr/store/shared';

import { PolicyDependenciesFacadeService } from '../../services/policy-dependencies-facade.service';
import { defaultPolicyPageState, PolicyPageState } from '../../utils';

@Component({
    selector: 'soldr-policy-page',
    templateUrl: './policy-page.component.html',
    styleUrls: ['./policy-page.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class PolicyPageComponent implements OnInit, AfterViewInit, OnDestroy {
    agents$ = this.policiesFacade.policyAgents$.pipe(
        map((agents) => this.dateFormatterService.formatToRelativeDate(agents, 'connected_date'))
    );
    agentsGridFiltrationByFields$ = this.policiesFacade.agentsGridFiltrationByFields$;
    agentsPage$ = this.policiesFacade.agentsPage$;
    agentsSearchValue$ = this.policiesFacade.agentsSearchValue$;
    agentsTotal$ = this.policiesFacade.agentsTotal$;
    dependencies$ = this.policyDependenciesFacadeService.$dependencies$;
    direction = Direction;
    events$ = this.policiesFacade.policyEvents$.pipe(
        map((events) => this.dateFormatterService.formatToAbsoluteLongWithSeconds(events, 'date'))
    );
    eventsGridFiltration$ = this.policiesFacade.eventsGridFiltration$;
    eventsGridFiltrationByFields$ = this.policiesFacade.eventsGridFiltrationByFields$;
    eventsPage$ = this.policiesFacade.eventsPage$;
    eventsSearchValue$ = this.policiesFacade.eventsSearchValue$;
    eventsTotal$ = this.policiesFacade.eventsTotal$;
    groups$ = this.policiesFacade.policyGroups$.pipe(
        map((groups) => this.dateFormatterService.formatToRelativeDate(groups, 'created_date'))
    );
    groupsGridFiltration$ = this.policiesFacade.groupsGridFiltration$;
    groupsGridFiltrationByFields$ = this.policiesFacade.groupsGridFiltrationByFields$;
    groupsPage$ = this.policiesFacade.groupsPage$;
    groupsSearchValue$ = this.policiesFacade.groupsSearchValue$;
    groupsTotal$ = this.policiesFacade.groupsTotal$;
    agentGridColumnFilterItems$ = this.policiesFacade.agentGridColumnFilterItems$;
    eventGridColumnFilterItems$ = this.policiesFacade.eventGridColumnFilterItems$;
    gridGroupColumnFilterItems$ = this.policiesFacade.gridGroupColumnFilterItems$;
    isCancelUpgradingAgent$ = this.policiesFacade.isCancelUpgradingAgent$;
    isLoadingAgents$ = this.policiesFacade.isLoadingAgents$;
    isLoadingEvents$ = this.policiesFacade.isLoadingEvents$;
    isLoadingGroups$ = this.policiesFacade.isLoadingGroups$;
    isLoadingLatestBinary$ = this.sharedFacade.isLoadingLatestAgentBinary$;
    isLoadingModules$ = this.sharedFacade.isLoadingAllModules$;
    isLoadingPolicy$ = this.policiesFacade.isLoadingPolicy$;
    isLoadingPolicyModules$ = this.policiesFacade.isLoadingModules$;
    isUpgradingAgents$ = this.policiesFacade.isUpgradingAgents$;
    language$ = this.languageService.current$;
    modules$ = this.policiesFacade.policyModules$;
    pageState: PolicyPageState;
    policy$ = this.policiesFacade.policy$;
    selectedAgent$ = this.policiesFacade.selectedAgent$;
    selectedGroup$ = this.policiesFacade.selectedPolicyGroup$;

    selectedModuleName: string;
    sidebarPositions = SidebarPositions;
    subscription = new Subscription();
    tabIndex = 0;
    viewModeEnum = ViewMode;

    @ViewChild('tabs') tabsEl: McTabGroup;

    constructor(
        private activatedRoute: ActivatedRoute,
        private dateFormatterService: DateFormatterService,
        private languageService: LanguageService,
        private pageTitleService: PageTitleService,
        private policiesFacade: PoliciesFacade,
        private policyDependenciesFacadeService: PolicyDependenciesFacadeService,
        private router: Router,
        private sharedFacade: SharedFacade,
        private transloco: TranslocoService,
        @Inject(PERMISSIONS_TOKEN) public permitted: ProxyPermission,
        @Inject(STATE_STORAGE_TOKEN) private stateStorage: StateStorage,
        @Inject(POLICY_LINKING_FACADE) private facade: LinkPolicyToGroupFacade<Policy, Group>
    ) {
        const paramsSubscription = this.activatedRoute.paramMap.subscribe(() => {
            this.refreshData();
        });
        this.subscription.add(paramsSubscription);
    }

    ngOnInit(): void {
        this.defineTitle();

        this.pageState = mergeDeep(
            defaultPolicyPageState(),
            (this.stateStorage.loadState('policy.view') as PolicyPageState) || {}
        );

        this.policiesFacade.fetchModules(this.activatedRoute.snapshot.params.hash as string);

        const policySubscription = this.policiesFacade.policy$.pipe(filter(Boolean)).subscribe(({ id }) => {
            this.policiesFacade.fetchEvents(id);
            this.policiesFacade.fetchAgents(id);
            this.policiesFacade.fetchGroups(id);
            this.policiesFacade.fetchAgentFilterItems();
            this.policiesFacade.fetchEventFilterItems();
            this.policiesFacade.fetchGroupFilterItems();
        });
        this.subscription.add(policySubscription);

        this.selectedModuleName = this.activatedRoute.snapshot.queryParams.moduleName;

        const entityForLinkingSubscription = this.policiesFacade.policy$.subscribe(this.facade.baseEntity$);
        this.subscription.add(entityForLinkingSubscription);

        const changeLinksSubscription = merge(
            this.policiesFacade.isLinkingPolicy$.pipe(pairwise()),
            this.policiesFacade.isUnlinkingPolicy$.pipe(pairwise())
        ).subscribe(([previous, current]) => {
            if (previous && !current) {
                this.policiesFacade.fetchPolicy(this.activatedRoute.snapshot.params.hash as string);
            }
        });
        this.subscription.add(changeLinksSubscription);
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
        this.policiesFacade.reset();
        this.subscription.unsubscribe();
    }

    refreshData() {
        this.policiesFacade.fetchPolicy(this.activatedRoute.snapshot.params.hash as string);
        this.sharedFacade.fetchAllModules();
        this.sharedFacade.fetchAllGroups();
        this.sharedFacade.fetchLatestAgentBinary();
    }

    refreshAgents(policyId: number) {
        this.policiesFacade.fetchAgents(policyId);
    }

    refreshDependencies() {
        this.policiesFacade.fetchPolicy(this.activatedRoute.snapshot.params.hash as string);
        this.sharedFacade.fetchAllModules();
    }

    refreshModules() {
        this.policiesFacade.fetchModules(this.activatedRoute.snapshot.params.hash as string);
    }

    refreshAfterChangeModuleState() {
        this.policiesFacade.fetchPolicy(this.activatedRoute.snapshot.params.hash as string);
    }

    saveLeftSidebarState(opened: boolean) {
        this.pageState.leftSidebar = { ...this.pageState.leftSidebar, opened };
    }

    saveLeftSidebarWidth($event: WidthChangeEvent) {
        if ($event.width !== '32px') {
            this.pageState.leftSidebar = { ...this.pageState.leftSidebar, width: $event.width };
        }
    }

    onSelectTab() {
        this.saveState();
    }

    eventsSearch(value: string) {
        this.policiesFacade.setEventsGridSearch(value);
    }

    eventsFilter(value: Filtration) {
        this.policiesFacade.setEventsGridFiltration(value);
    }

    loadNextEventsPage(id: number, page: number) {
        this.policiesFacade.fetchEvents(id, page);
    }

    eventsResetFiltration() {
        this.policiesFacade.resetEventsFiltration();
    }

    eventsSort(sorting: Sorting) {
        this.policiesFacade.setEventsGridSorting(sorting);
    }

    selectTag(tag: string) {
        this.router.navigate(['/policies'], {
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

    onAfterDelete() {
        this.router.navigate(['/policies']);
    }

    agentsSearch(value: string) {
        this.policiesFacade.setAgentsGridSearch(value);
    }

    agentsFilter(value: Filtration) {
        this.policiesFacade.setAgentsGridFiltration(value);
    }

    agentsSort(sorting: Sorting) {
        this.policiesFacade.setAgentsGridSorting(sorting);
    }

    resetAgentsFiltration() {
        this.policiesFacade.resetAgentsFiltration();
    }

    loadNextAgentsPage(id: number, page: number) {
        this.policiesFacade.fetchAgents(id, page);
    }

    agentsSelect(agent: Agent) {
        this.policiesFacade.selectAgent(agent?.id);
    }

    groupsSearch(value: string) {
        this.policiesFacade.setGroupsGridSearch(value);
    }

    groupsFilter(value: Filtration) {
        this.policiesFacade.setGroupsGridFiltration(value);
    }

    groupsSort(sorting: Sorting) {
        this.policiesFacade.setGroupsGridSorting(sorting);
    }

    resetGroupsFiltration() {
        this.policiesFacade.resetGroupsFiltration();
    }

    loadNextGroupsPage(id: number, page: number) {
        this.policiesFacade.fetchGroups(id, page);
    }

    groupsSelect(group: Group) {
        this.policiesFacade.selectPolicyGroup(group?.id);
    }

    onSetAgentsTag(tag: string) {
        this.policiesFacade.setAgentsGridFiltrationByTag(tag);
    }

    onSetGroupsTag(tag: string) {
        this.policiesFacade.setGroupsGridFiltrationByTag(tag);
    }

    onSelectModule(module: EntityModule) {
        this.selectedModuleName = module?.info.name;
        this.saveState();
    }

    afterUpgradeAgent(agent: Agent) {
        this.policiesFacade.updateAgentData(agent);
    }

    upgradeAgents(event: { agents: Agent[]; version: string }) {
        this.policiesFacade.upgradeAgents(event.agents, event.version);
    }

    cancelUpgradeAgent(event: { hash: string; task: AgentUpgradeTask }) {
        this.policiesFacade.cancelUpgradeAgent(event.hash, event.task);
    }

    updateModuleConfig(module: ModelsModuleA) {
        this.policiesFacade.updateModuleConfig(module);
    }

    refreshGroups(policyId: number) {
        this.policiesFacade.fetchGroups(policyId);
    }

    refreshEvents(policyId: number) {
        this.policiesFacade.fetchEvents(policyId);
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
            policyNameTitle,
            this.transloco.selectTranslate('Policies.PageTitle.Text.Policies', {}, 'policies'),
            this.sharedFacade.selectedServiceName$,
            this.transloco.selectTranslate('Shared.Pseudo.PageTitle.ApplicationName', {}, 'shared')
        ])
            .pipe(map((segments) => segments.filter(Boolean)))
            .subscribe((segments) => this.pageTitleService.setTitle(segments));

        this.subscription.add(titlesSubscription);
    }
}
