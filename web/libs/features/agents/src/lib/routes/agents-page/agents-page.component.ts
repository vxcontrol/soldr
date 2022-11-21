import { ChangeDetectionStrategy, Component, Inject, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { ActivatedRoute, Params, Router } from '@angular/router';
import { TranslocoService } from '@ngneat/transloco';
import { PopUpPlacements, ThemePalette } from '@ptsecurity/mosaic/core';
import { SidebarPositions } from '@ptsecurity/mosaic/sidebar';
import { Direction } from '@ptsecurity/mosaic/splitter';
import {
    BehaviorSubject,
    combineLatest,
    combineLatestWith,
    filter,
    map,
    Observable,
    pairwise,
    skipWhile,
    startWith,
    Subscription,
    take
} from 'rxjs';

import { ModelsGroupItemLocale } from '@soldr/api';
import { PERMISSIONS_TOKEN } from '@soldr/core';
import { AgentsExporterService } from '@soldr/features/agents';
import { Agent, AgentUpgradeTask, canUpgradeAgent, Group } from '@soldr/models';
import {
    DateFormatterService,
    Filtration,
    GridColumnFilterItem,
    GridComponent,
    LanguageService,
    ListItem,
    PageTitleService,
    STATE_STORAGE_TOKEN,
    Selection,
    Sorting,
    StateStorage,
    ViewMode,
    WidthChangeEvent,
    sortModules,
    sortTags,
    mergeDeep,
    Filter,
    ProxyPermission
} from '@soldr/shared';
import { AgentListFacade } from '@soldr/store/agents';
import { SharedFacade } from '@soldr/store/shared';
import { TagDomain, TagsFacade } from '@soldr/store/tags';

import { AgentsPageState, defaultAgentsPageState, defaultFilters, gridFilters } from '../../utils';

const FIELD_FOR_TRANSLATE = ['status', 'os', 'auth_status'];

@Component({
    selector: 'soldr-agents-page',
    templateUrl: './agents-page.component.html',
    styleUrls: ['./agents-page.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class AgentsPageComponent implements OnInit, OnDestroy {
    agent$: Observable<Agent> = this.agentListFacade.agent$;
    agentModules$ = this.agentListFacade.agentModules$;
    agents$ = this.agentListFacade.agents$;
    allGroups$ = this.sharedFacade.allGroups$;
    canBlockSelectedAgents$: Observable<boolean>;
    canUpgradeSelectedAgents$: Observable<boolean>;
    deleteError$ = this.agentListFacade.deleteError$;
    direction = Direction;
    emptyText$: Observable<string>;
    filters$ = this.agentListFacade.filters$;
    dataGrid$: Observable<Agent[]>;
    gridColumnsFilters: { [field: string]: GridColumnFilterItem[] } = gridFilters();
    gridFiltration$ = this.agentListFacade.gridFiltration$;
    gridFiltrationByField$ = this.agentListFacade.gridFiltrationByField$;
    groupsTags$: Observable<ListItem[]>;
    isBlockingAgents$ = this.agentListFacade.isBlockingAgents$;
    isDeletingAgent$ = this.agentListFacade.isDeletingAgents$;
    isDeletingFromGroup$ = this.agentListFacade.isDeletingFromGroup$;
    isLoadingAgent$ = this.agentListFacade.isLoadingAgent$;
    isLoadingAgents$ = this.agentListFacade.isLoading$;
    isLoadingAllGroups$ = this.sharedFacade.isLoadingAllGroups$;
    isMovingAgents$ = this.agentListFacade.isMovingAgents$;
    isUpdatingAgent$ = this.agentListFacade.isUpdatingAgent$;
    isUpgradingAgents$ = this.agentListFacade.isUpgradingAgents$;
    isCancelUpgradingAgent$ = this.agentListFacade.isCancelUpgradingAgent$;
    language$ = this.languageService.current$;
    latestAgentBinaryVersion$ = this.sharedFacade.latestAgentBinaryVersion$;
    moveToGroupError$ = this.agentListFacade.moveToGroupError$;
    page$ = this.agentListFacade.page$;
    pageState: AgentsPageState;
    placement = PopUpPlacements;
    search$ = this.agentListFacade.search$;
    searchGroupsTags$ = new BehaviorSubject<string>('');
    selected$: Observable<Agent[]> = this.agentListFacade.selectedAgents$;
    selectedFilterId$: Observable<string | undefined>;
    selectedGroupsTags$ = this.sharedFacade.selectedGroupTags$;
    selection = Selection;
    sidebarPositions = SidebarPositions;
    sortModules = sortModules;
    sortTags = sortTags;
    sorting$ = this.agentListFacade.sorting$;
    subscription = new Subscription();
    themePalette = ThemePalette;
    total$ = this.agentListFacade.total$;
    updateError$ = this.agentListFacade.updateError$;
    viewMode = ViewMode;

    @ViewChild('grid') private grid: GridComponent;

    constructor(
        private activatedRoute: ActivatedRoute,
        private agentListFacade: AgentListFacade,
        private agentsExporterService: AgentsExporterService,
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
        const translatedFilters = defaultFilters().map(
            (filter): Filter => ({ ...filter, label: transloco.translate(filter.label) })
        );
        this.agentListFacade.updateFilters(translatedFilters);
        this.agentListFacade.fetchFiltersCounters();
        this.agentListFacade.fetchVersions();
        this.sharedFacade.fetchAllModules();
        this.sharedFacade.fetchAllGroups();
        this.tagsFacade.fetchTags(TagDomain.Groups);
        this.sharedFacade.fetchLatestAgentBinary();
    }

    ngOnInit(): void {
        this.defineTitle();

        this.initGroupsAndFiltersRx();
        this.initPageStateRx();
        this.initDataRx();
        this.initGridRx();

        const isMovingAgentsSubscription = this.isMovingAgents$
            .pipe(pairwise(), combineLatestWith(this.moveToGroupError$))
            .subscribe(([[oldValue, newValue], moveToGroupError]) => {
                if (oldValue && !newValue && !moveToGroupError) {
                    this.sharedFacade.fetchAllGroups();
                }
            });
        this.subscription.add(isMovingAgentsSubscription);

        this.pageState = mergeDeep(
            defaultAgentsPageState(),
            (this.stateStorage.loadState('agents.list') as AgentsPageState) || {}
        );
    }

    ngOnDestroy(): void {
        this.agentListFacade.reset();
        this.sharedFacade.resetFilterByTags();
        this.subscription.unsubscribe();
    }

    onSearchGroupsTags(value: string): void {
        this.searchGroupsTags$.next(value);
    }

    onSelectGroupsTags(tags: string[]): void {
        this.sharedFacade.setFilterByTags(tags);
        this.sharedFacade.fetchAllGroups();
    }

    onSelectFilter(filterId: string): void {
        this.agentListFacade.selectFilter(filterId);
        this.agentListFacade.fetchFiltersCounters();
        this.sharedFacade.fetchAllGroups();
    }

    onSelectGroup(groupId: string) {
        this.agentListFacade.selectGroup(groupId);
        this.agentListFacade.fetchFiltersCounters();
        this.sharedFacade.fetchAllGroups();
    }

    onGridFilter(filtration: Filtration) {
        this.agentListFacade.setGridFiltration(filtration);
    }

    onGridSearch(value: string) {
        this.agentListFacade.setGridSearch(value);
    }

    onResetFiltration() {
        this.agentListFacade.resetFiltration();
    }

    onGridSort(sorting: Sorting) {
        this.agentListFacade.setGridSorting(sorting);
    }

    onGridSelectRows(agents: Agent[]) {
        this.agentListFacade.resetAgentErrors();
        this.agentListFacade.selectAgents(agents);
    }

    onExport($event: { columns: string[]; selected?: any[] }) {
        this.agentsExporterService.export($event.columns, $event.selected as Agent[]);
    }

    upgradeAgents(selected: Agent[], latestAgentBinaryVersion: string) {
        this.agentListFacade.isUpgradingAgents$.pipe(pairwise(), take(2)).subscribe(([oldValue, newValue]) => {
            if (oldValue && !newValue) {
                this.agentListFacade.updateAgentsData(selected);
            }
        });
        this.agentListFacade.upgradeAgents(selected, latestAgentBinaryVersion);
    }

    blockAgents(selected: Agent[]) {
        this.agentListFacade.blockAgents(selected.map(({ id }) => id));
    }

    setTag(tag: string) {
        this.agentListFacade.setGridFiltrationByTag(tag);
    }

    saveLeftSidebarState(opened: boolean) {
        this.pageState.leftSidebar = { ...this.pageState.leftSidebar, opened };
    }

    saveLeftSidebarWidth($event: WidthChangeEvent) {
        if ($event.width !== '32px') {
            this.pageState.leftSidebar = { ...this.pageState.leftSidebar, width: $event.width };
        }
    }

    saveRightSidebarState(opened: boolean) {
        this.pageState.rightSidebar = { ...this.pageState.rightSidebar, opened };
    }

    saveRightSidebarWidth($event: WidthChangeEvent) {
        if ($event.width !== '32px') {
            this.pageState.rightSidebar = { ...this.pageState.rightSidebar, width: $event.width };
        }
    }

    refreshData(page?: number) {
        this.agentListFacade.fetchAgentsPage(page);
    }

    refreshGroups() {
        this.sharedFacade.fetchAllGroups();
    }

    afterUpgradeAgent(agent: Agent) {
        this.agentListFacade.updateAgentsData([agent]);
    }

    moveToGroups(event: { ids: number[]; groupId: number }) {
        this.agentListFacade.moveToGroups(event.ids, event.groupId);
    }

    moveToNewGroups(event: { ids: number[]; groupName: string }) {
        this.agentListFacade.moveToNewGroups(event.ids, event.groupName);
    }

    delete(ids: number[]) {
        this.agentListFacade.deleteAgents(ids);
    }

    update(event: Agent) {
        this.agentListFacade.updateAgent(event);
    }

    upgrade(event: { agents: Agent[]; version: string }) {
        this.agentListFacade.upgradeAgents(event.agents, event.version);
    }

    cancelUpgrade(event: { hash: string; task: AgentUpgradeTask }) {
        this.agentListFacade.cancelUpgradeAgent(event.hash, event.task);
    }

    private defineTitle() {
        const titlesSubscription = combineLatest([
            this.transloco.selectTranslate('Agents.PageTitle.Text.Agents', {}, 'agents'),
            this.sharedFacade.selectedServiceName$,
            this.transloco.selectTranslate('Shared.Pseudo.PageTitle.ApplicationName', {}, 'shared')
        ])
            .pipe(map((segments) => segments.filter(Boolean)))
            .subscribe((segments) => this.pageTitleService.setTitle(segments));

        this.subscription.add(titlesSubscription);
    }

    private initGridRx() {
        this.agentListFacade.fetchFilterItems();

        const gridFilterItemsSubscription: Subscription = this.agentListFacade.gridColumnFilterItems$.subscribe(
            (gridFilters) => {
                this.gridColumnsFilters = {
                    status: [...this.gridColumnsFilters.status],
                    auth_status: [...this.gridColumnsFilters.auth_status],
                    ...gridFilters
                };
            }
        );
        this.subscription.add(gridFilterItemsSubscription);

        FIELD_FOR_TRANSLATE.forEach(
            (field) =>
                (this.gridColumnsFilters[field] = this.gridColumnsFilters[field].map((filterItem) => ({
                    ...filterItem,
                    label: this.transloco.translate(filterItem.label)
                })))
        );

        const selectedGroup$ = combineLatest([
            this.agentListFacade.selectedGroupId$,
            this.sharedFacade.allGroups$
        ]).pipe(
            map(([groupId, allGroups]: [string, Group[]]) => allGroups.find((group) => group.id === parseInt(groupId)))
        );

        this.emptyText$ = combineLatest([
            this.agentListFacade.agents$,
            this.agentListFacade.search$,
            this.agentListFacade.selectedFilterId$,
            selectedGroup$,
            this.languageService.current$
        ]).pipe(
            map(([agents, search, filterId, group, lang]) => {
                if (agents.length === 0) {
                    if (search) return this.transloco.translate('agents.Agents.AgentsList.Text.NoAgentsOnSearch');
                    if (filterId && filterId !== 'all')
                        return this.transloco.translate('agents.Agents.AgentsList.Text.NoAgentsInFilter', {
                            filter: filterId
                        });
                    if (group)
                        return this.transloco.translate('agents.Agents.AgentsList.Text.NoAgentsInGroup', {
                            group: group.info.name[lang as keyof ModelsGroupItemLocale]
                        });

                    return this.transloco.translate('agents.Agents.AgentsList.Text.NoAgents');
                }

                return undefined;
            })
        );
    }

    private initDataRx() {
        this.dataGrid$ = this.agents$.pipe(
            map((agents) => this.dateFormatterService.formatToRelativeDate(agents, 'connected_date'))
        );
        this.agentListFacade.isRestored$.pipe(filter(Boolean), take(1)).subscribe(() => {
            this.agentListFacade.fetchAgentsPage();
        });

        this.canUpgradeSelectedAgents$ = combineLatest([
            this.agentListFacade.selectedAgents$,
            this.sharedFacade.latestAgentBinaryVersion$
        ]).pipe(
            map(([agents, latestBinaryVersion]) => agents.some((agent) => canUpgradeAgent(agent, latestBinaryVersion)))
        );

        this.canBlockSelectedAgents$ = this.agentListFacade.selectedAgents$.pipe(
            map((agents) => agents.filter(({ auth_status }) => auth_status !== 'blocked').length > 0)
        );
    }

    private initPageStateRx() {
        // restore state
        combineLatest([this.agentListFacade.isInitialized$, this.sharedFacade.initializedGroups$])
            .pipe(
                skipWhile(([isAgentInitialized, isGroupsInitialized]) => !isAgentInitialized || !isGroupsInitialized),
                take(1)
            )
            .subscribe(() => {
                this.agentListFacade.restoreState();
            });

        // save state
        const saveStateSubscription = combineLatest([
            this.agentListFacade.isRestored$,
            this.agentListFacade.selectedFilterId$,
            this.agentListFacade.selectedGroupId$,
            this.agentListFacade.gridFiltration$,
            this.agentListFacade.search$,
            this.agentListFacade.sorting$
        ])
            .pipe(skipWhile(([restored]) => !restored))
            .subscribe(([, filterId, groupId, filtration, search, sorting]) => {
                const queryParams: Params = {
                    filterId,
                    groupId,
                    filtration: filtration.length > 0 ? JSON.stringify(filtration) : undefined,
                    search: search || undefined,
                    sort: Object.keys(sorting).length > 0 ? JSON.stringify(sorting) : undefined
                };

                this.router.navigate([], {
                    relativeTo: this.activatedRoute,
                    queryParams,
                    queryParamsHandling: 'merge',
                    replaceUrl: true
                });
            });
        this.subscription.add(saveStateSubscription);
    }

    private initGroupsAndFiltersRx() {
        // selected filter in groups and filters
        this.selectedFilterId$ = combineLatest([
            this.agentListFacade.selectedFilterId$,
            this.agentListFacade.selectedGroupId$
        ]).pipe(map(([filterId, groupId]) => filterId || groupId));

        // filter groups by tags
        this.groupsTags$ = combineLatest([
            this.tagsFacade.groupsTags$,
            this.searchGroupsTags$.pipe(startWith(''))
        ]).pipe(
            map(([tags, searchValue]) =>
                tags
                    ?.filter((tag) => tag.toLocaleLowerCase().includes(searchValue.toLocaleLowerCase()))
                    .map((tag: string) => ({ label: tag, value: tag } as ListItem))
            )
        );
    }
}
