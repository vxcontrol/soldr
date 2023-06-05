import { AfterViewInit, ChangeDetectionStrategy, Component, Inject, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslocoService } from '@ngneat/transloco';
import { SidebarPositions } from '@ptsecurity/mosaic/sidebar';
import { Direction } from '@ptsecurity/mosaic/splitter';
import { McTabGroup } from '@ptsecurity/mosaic/tabs';
import { combineLatest, filter, map, merge, pairwise, Subscription, switchMap } from 'rxjs';

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
    ViewMode
} from '@soldr/shared';
import { GroupsFacade } from '@soldr/store/groups';
import { SharedFacade } from '@soldr/store/shared';

import { GroupDependenciesFacadeService, LinkPolicyFacadeService } from '../../services';
import { defaultGroupPageState, GroupPageState } from '../../utils';

@Component({
    selector: 'soldr-group-page',
    templateUrl: './group-page.component.html',
    styleUrls: ['./group-page.component.scss'],
    providers: [{ provide: POLICY_LINKING_FACADE, useClass: LinkPolicyFacadeService }],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class GroupPageComponent implements OnInit, AfterViewInit, OnDestroy {
    @ViewChild('tabs') tabsEl: McTabGroup;

    group$ = this.groupsFacade.group$;
    language$ = this.languageService.current$;
    modules$ = this.groupsFacade.groupModules$;
    agents$ = this.groupsFacade.groupAgents$.pipe(
        map((agents) => this.dateFormatterService.formatToRelativeDate(agents, 'connected_date'))
    );
    events$ = this.groupsFacade.groupEvents$.pipe(
        map((events) => this.dateFormatterService.formatToAbsoluteLongWithSeconds(events, 'date'))
    );
    policies$ = this.groupsFacade.groupPolicies$.pipe(
        map((policy) => this.dateFormatterService.formatToRelativeDate(policy, 'created_date'))
    );
    dependencies$ = this.groupDependenciesFacadeService.dependencies$;
    agentsPage$ = this.groupsFacade.agentsPage$;
    eventsPage$ = this.groupsFacade.eventsPage$;
    policiesPage$ = this.groupsFacade.policiesPage$;
    agentsGridFiltrationByFields$ = this.groupsFacade.agentsGridFiltrationByFields$;
    eventsGridFiltrationByFields$ = this.groupsFacade.eventsGridFiltrationByFields$;
    policiesGridFiltrationByFields$ = this.groupsFacade.policiesGridFiltrationByFields$;
    selectedAgent$ = this.groupsFacade.selectedAgent$;
    selectedPolicy$ = this.groupsFacade.selectedPolicy$;
    agentsTotal$ = this.groupsFacade.agentsTotal$;
    eventsTotal$ = this.groupsFacade.eventsTotal$;
    policiesTotal$ = this.groupsFacade.policiesTotal$;
    agentsSearchValue$ = this.groupsFacade.agentsSearchValue$;
    eventsSearchValue$ = this.groupsFacade.eventsSearchValue$;
    policiesSearchValue$ = this.groupsFacade.policiesSearchValue$;
    agentGridColumnFilterItems$ = this.groupsFacade.agentGridColumnFilterItems$;
    policyGridColumnFilterItems$ = this.groupsFacade.policyGridColumnFilterItems$;
    eventGridColumnFilterItems$ = this.groupsFacade.eventGridColumnFilterItems$;
    isCancelUpgradingAgent$ = this.groupsFacade.isCancelUpgradingAgent$;
    isLoadingAgents$ = this.groupsFacade.isLoadingAgents$;
    isLoadingGroup$ = this.groupsFacade.isLoadingGroup$;
    isLoadingModules$ = this.sharedFacade.isLoadingAllModules$;
    isLoadingPolicies$ = this.groupsFacade.isLoadingPolicies$;
    isLoadingEvents$ = this.groupsFacade.isLoadingEvents$;
    isUpgradingAgents$ = this.groupsFacade.isUpgradingAgents$;

    direction = Direction;
    sidebarPositions = SidebarPositions;
    pageState: GroupPageState;
    tabIndex = 0;
    selectedModuleName: string;
    subscription = new Subscription();
    viewModeEnum = ViewMode;

    constructor(
        private activatedRoute: ActivatedRoute,
        private dateFormatterService: DateFormatterService,
        private groupDependenciesFacadeService: GroupDependenciesFacadeService,
        private groupsFacade: GroupsFacade,
        private languageService: LanguageService,
        private pageTitleService: PageTitleService,
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
        this.sharedFacade.fetchAllGroups();
    }

    ngOnInit(): void {
        this.defineTitle();

        this.pageState = mergeDeep(
            defaultGroupPageState(),
            (this.stateStorage.loadState('group.view') as GroupPageState) || {}
        );
        this.groupsFacade.fetchGroup(this.activatedRoute.snapshot.params.hash as string);
        const groupSubscription = this.groupsFacade.group$.pipe(filter(Boolean)).subscribe(({ id }) => {
            this.groupsFacade.fetchAgents(id);
            this.groupsFacade.fetchPolicies(id);
            this.groupsFacade.fetchEvents(id);
            this.groupsFacade.fetchAgentFilterItems();
            this.groupsFacade.fetchPolicyFilterItems();
            this.groupsFacade.fetchEventFilterItems();
        });
        this.subscription.add(groupSubscription);

        this.selectedModuleName = this.activatedRoute.snapshot.queryParams.moduleName;

        const entityForLinkingSubscription = this.groupsFacade.group$.subscribe(this.facade.baseEntity$);
        this.subscription.add(entityForLinkingSubscription);

        const changeLinksSubscription = merge(
            this.groupsFacade.isLinkingGroup$.pipe(pairwise()),
            this.groupsFacade.isUnlinkingGroup$.pipe(pairwise())
        ).subscribe(([previous, current]) => {
            if (previous && !current) {
                this.groupsFacade.fetchGroup(this.activatedRoute.snapshot.params.hash as string);
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
        this.groupsFacade.reset();
        this.subscription.unsubscribe();
    }

    refreshData() {
        this.groupsFacade.fetchGroup(this.activatedRoute.snapshot.params.hash as string);
        this.sharedFacade.fetchAllModules();
        this.sharedFacade.fetchAllPolicies();
        this.sharedFacade.fetchLatestAgentBinary();
    }

    refreshAgents(groupId: number) {
        this.groupsFacade.fetchAgents(groupId);
    }

    selectTag(tag: string) {
        this.router.navigate(['/groups'], {
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
        this.router.navigate(['/groups']);
    }

    agentsSelect(agent: Agent) {
        this.groupsFacade.selectAgent(agent?.id);
    }

    policySelect(policy: Policy) {
        this.groupsFacade.selectPolicy(policy?.id);
    }

    agentsSearch(value: string) {
        this.groupsFacade.setAgentsGridSearch(value);
    }

    policySearch(value: string) {
        this.groupsFacade.setPoliciesGridSearch(value);
    }

    agentsFilter(value: Filtration) {
        this.groupsFacade.setAgentsGridFiltration(value);
    }

    agentsSort(sorting: Sorting) {
        this.groupsFacade.setAgentsGridSorting(sorting);
    }

    policiesSort(sorting: Sorting) {
        this.groupsFacade.setPoliciesGridSorting(sorting);
    }

    policyFilter(value: Filtration) {
        this.groupsFacade.setPoliciesGridFiltration(value);
    }

    resetAgentsFiltration() {
        this.groupsFacade.resetAgentsFiltration();
    }

    resetPolicyFiltration() {
        this.groupsFacade.resetPoliciesFiltration();
    }

    resetEventsFiltration() {
        this.groupsFacade.resetEventsFiltration();
    }

    loadNextAgentsPage(id: number, page: number) {
        this.groupsFacade.fetchAgents(id, page);
    }

    loadNextPolicyPage(id: number, page: number) {
        this.groupsFacade.fetchPolicies(id, page);
    }

    onSetAgentsTag(tag: string) {
        this.groupsFacade.setAgentsGridFiltrationByTag(tag);
    }

    onSetPolicyTag(tag: string) {
        this.groupsFacade.setPoliciesGridFiltrationByTag(tag);
    }

    onSelectTab() {
        this.saveState();
    }

    eventsSearch(value: string) {
        this.groupsFacade.setEventsGridSearch(value);
    }

    eventsFilter(value: Filtration) {
        this.groupsFacade.setEventsGridFiltration(value);
    }

    loadNextEventsPage(id: number, page: number) {
        this.groupsFacade.fetchEvents(id, page);
    }

    eventsSort(sorting: Sorting) {
        this.groupsFacade.setEventsGridSorting(sorting);
    }

    onSelectModule(module: EntityModule) {
        this.selectedModuleName = module?.info.name;
        this.saveState();
    }

    upgradeAgents(event: { agents: Agent[]; version: string }) {
        this.groupsFacade.upgradeAgents(event.agents, event.version);
    }

    cancelUpgradeAgent(event: { hash: string; task: AgentUpgradeTask }) {
        this.groupsFacade.cancelUpgradeAgent(event.hash, event.task);
    }

    afterUpgradeAgent(agent: Agent) {
        this.groupsFacade.updateAgentData(agent);
    }

    refreshPolicies(groupId: number) {
        this.groupsFacade.fetchPolicies(groupId);
    }

    refreshEvents(groupId: number) {
        this.groupsFacade.fetchEvents(groupId);
    }

    refreshDependencies() {
        this.groupsFacade.fetchGroup(this.activatedRoute.snapshot.params.hash as string);
        this.sharedFacade.fetchAllModules();
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
        const groupNameTitle = combineLatest([this.group$, this.language$]).pipe(
            filter(([group]) => Boolean(group)),
            switchMap(([group, lang]) =>
                this.transloco.selectTranslate(
                    'Groups.PageTitle.Text.Group',
                    { group: group.info.name[lang] },
                    'groups'
                )
            )
        );
        const titlesSubscription = combineLatest([
            groupNameTitle,
            this.transloco.selectTranslate('Groups.PageTitle.Text.Groups', {}, 'groups'),
            this.sharedFacade.selectedServiceName$,
            this.transloco.selectTranslate('Shared.Pseudo.PageTitle.ApplicationName', {}, 'shared')
        ])
            .pipe(map((segments) => segments.filter(Boolean)))
            .subscribe((segments) => this.pageTitleService.setTitle(segments));

        this.subscription.add(titlesSubscription);
    }
}
