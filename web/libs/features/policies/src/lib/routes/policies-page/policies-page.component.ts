import { ChangeDetectionStrategy, Component, Inject, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { ActivatedRoute, Params, Router } from '@angular/router';
import { TranslocoService } from '@ngneat/transloco';
import { PopUpPlacements } from '@ptsecurity/mosaic/core';
import { SidebarPositions } from '@ptsecurity/mosaic/sidebar';
import { Direction } from '@ptsecurity/mosaic/splitter';
import {
    combineLatest,
    filter,
    first,
    map,
    merge,
    Observable,
    pairwise,
    ReplaySubject,
    skipWhile,
    startWith,
    Subscription,
    take
} from 'rxjs';

import { ModelsGroupItemLocale } from '@soldr/api';
import { PERMISSIONS_TOKEN } from '@soldr/core';
import { Group, Policy } from '@soldr/models';
import {
    Filtration,
    GridColumnFilterItem,
    LanguageService,
    ListItem,
    osList,
    sortModules,
    STATE_STORAGE_TOKEN,
    StateStorage,
    WidthChangeEvent,
    sortTags,
    sortGroups,
    Sorting,
    ViewMode,
    PageTitleService,
    DateFormatterService,
    mergeDeep,
    Filter,
    POLICY_LINKING_FACADE,
    LinkPolicyToGroupFacade,
    ProxyPermission,
    GridComponent
} from '@soldr/shared';
import { PoliciesFacade } from '@soldr/store/policies';
import { SharedFacade } from '@soldr/store/shared';
import { TagDomain, TagsFacade } from '@soldr/store/tags';

import { PoliciesExporterService } from '../../services/policies-exporter.service';
import { defaultPoliciesPageState, PoliciesPageState, defaultFilters } from '../../utils';

@Component({
    selector: 'soldr-policies-page',
    templateUrl: './policies-page.component.html',
    styleUrls: ['./policies-page.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class PoliciesPageComponent implements OnInit, OnDestroy {
    @ViewChild('policiesGrid') policiesGridComponent: GridComponent;

    dataGrid$: Observable<Policy[]>;
    direction = Direction;
    emptyText$: Observable<string>;
    filters$ = this.policiesFacade.filters$;
    gridColumnsFilters: { [field: string]: GridColumnFilterItem[] } = {
        modules: [],
        tags: [],
        groups: [],
        os: [...osList]
    };
    gridFiltration$ = this.policiesFacade.gridFiltration$;
    gridFiltrationByField$ = this.policiesFacade.gridFiltrationByField$;
    groupsTags$: Observable<ListItem[]>;
    isLoading$ = this.policiesFacade.isLoadingPolicies$;
    isLoadingPolicyInfo$ = combineLatest([
        this.policiesFacade.isLoadingPolicy$,
        this.policiesFacade.isLoadingModules$
    ]).pipe(map(([a, b]) => a || b));
    language$ = this.languageService.current$;
    page$ = this.policiesFacade.page$;
    pageState: PoliciesPageState;
    placement = PopUpPlacements;
    policies$ = this.policiesFacade.policies$;
    policyModules$ = this.policiesFacade.policyModules$;
    search$ = this.policiesFacade.search$;
    searchGroupsTags$ = new ReplaySubject<string>(1);
    selected$ = this.policiesFacade.selectedPolicies$;
    selectedFilterId$: Observable<string | undefined>;
    selectedGroupsTags$ = this.sharedFacade.selectedGroupTags$;
    sidebarPositions = SidebarPositions;
    sortGroups = sortGroups;
    sortModules = sortModules;
    sortTags = sortTags;
    sorting$ = this.policiesFacade.sorting$;
    subscription = new Subscription();
    total$ = this.policiesFacade.total$;
    viewModeEnum = ViewMode;

    constructor(
        private activatedRoute: ActivatedRoute,
        private dateFormatterService: DateFormatterService,
        private languageService: LanguageService,
        private pageTitleService: PageTitleService,
        private policiesFacade: PoliciesFacade,
        private router: Router,
        private sharedFacade: SharedFacade,
        private tagsFacade: TagsFacade,
        private transloco: TranslocoService,
        private policiesExporter: PoliciesExporterService,
        @Inject(PERMISSIONS_TOKEN) public permitted: ProxyPermission,
        @Inject(STATE_STORAGE_TOKEN) private stateStorage: StateStorage,
        @Inject(POLICY_LINKING_FACADE) private facade: LinkPolicyToGroupFacade<Policy, Group>
    ) {
        const translatedFilters = defaultFilters().map(
            (filter): Filter => ({ ...filter, label: transloco.translate(filter.label) })
        );
        this.policiesFacade.updateFilters(translatedFilters);
        this.policiesFacade.fetchFiltersCounters();
        this.sharedFacade.fetchAllModules();
        this.tagsFacade.fetchTags(TagDomain.Groups);
    }

    ngOnInit(): void {
        this.defineTitle();
        this.initGroupsAndFiltersRx();
        this.initPageStateRx();
        this.initDataRx();
        this.initGridRx();

        this.pageState = mergeDeep(
            defaultPoliciesPageState(),
            (this.stateStorage.loadState('policies.list') as PoliciesPageState) || {}
        );
    }

    ngOnDestroy(): void {
        this.policiesFacade.reset();
        this.sharedFacade.resetFilterByTags();
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

    saveRightSidebarState(opened: boolean) {
        this.pageState.rightSidebar = { ...this.pageState.rightSidebar, opened };
    }

    saveRightSidebarWidth($event: WidthChangeEvent) {
        if ($event.width !== '32px') {
            this.pageState.rightSidebar = { ...this.pageState.rightSidebar, width: $event.width };
        }
    }

    onSearchGroupsTags(value: string): void {
        this.searchGroupsTags$.next(value);
    }

    onSelectGroupsTags(tags: string[]): void {
        this.sharedFacade.setFilterByTags(tags);
        this.sharedFacade.fetchAllGroups();
    }

    onSelectFilter(filterId: string) {
        this.policiesFacade.selectFilter(filterId);
        this.policiesFacade.fetchFiltersCounters();
        this.sharedFacade.fetchAllGroups(true);
    }

    onSelectGroup(groupId: string) {
        this.policiesFacade.selectGroup(groupId);
        this.policiesFacade.fetchFiltersCounters();
        this.sharedFacade.fetchAllGroups(true);
    }

    onGridSearch(value: string) {
        this.policiesFacade.setGridSearch(value);
    }

    onGridSort(sorting: Sorting | Record<never, any>) {
        this.policiesFacade.setGridSorting(sorting);
    }

    onGridSelectRows(policies: Policy[]) {
        this.policiesFacade.resetPolicyErrors();
        this.policiesFacade.selectPolicies(policies);
        this.policiesFacade.fetchModules(policies[0].hash);
    }

    onResetFiltration() {
        this.policiesFacade.resetFiltration();
    }

    loadNextPage(page: number) {
        this.policiesFacade.fetchPoliciesPage(page);
    }

    setTag(tag: string) {
        this.policiesFacade.setGridFiltrationByTag(tag);
    }

    onGridFilter(filtration: Filtration) {
        this.policiesFacade.setGridFiltration(filtration);
    }

    refreshSelected() {
        this.policiesFacade.createdPolicy$.pipe(filter(Boolean), first()).subscribe(({ id }) =>
            setTimeout(() => {
                this.policiesGridComponent.gridApi.getRowNode(`${id}`).setSelected(true);
                this.policiesFacade.resetCreatedPolicy();
            })
        );
    }

    refreshData() {
        this.policiesFacade.fetchPoliciesPage(1);
    }

    onExport($event: { selected?: any[]; columns: string[] }) {
        this.policiesExporter.export($event.columns, $event.selected as Policy[]);
    }

    private defineTitle() {
        const titlesSubscription = combineLatest([
            this.transloco.selectTranslate('Policies.PageTitle.Text.Policies', {}, 'policies'),
            this.sharedFacade.selectedServiceName$,
            this.transloco.selectTranslate('Shared.Pseudo.PageTitle.ApplicationName', {}, 'shared')
        ])
            .pipe(map((segments) => segments.filter(Boolean)))
            .subscribe((segments) => this.pageTitleService.setTitle(segments));

        this.subscription.add(titlesSubscription);
    }

    private initGroupsAndFiltersRx() {
        // selected filter in groups and filters
        this.selectedFilterId$ = combineLatest([
            this.policiesFacade.selectedFilterId$,
            this.policiesFacade.selectedGroupId$
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

    private initGridRx() {
        this.policiesFacade.fetchFilterItems();

        const selectedGroup$ = combineLatest([this.policiesFacade.selectedGroupId$, this.sharedFacade.allGroups$]).pipe(
            map(([groupId, allGroups]: [string, Group[]]) => allGroups.find((group) => group.id === parseInt(groupId)))
        );

        this.emptyText$ = combineLatest([
            this.policiesFacade.policies$,
            this.policiesFacade.search$,
            this.policiesFacade.selectedFilterId$,
            selectedGroup$,
            this.languageService.current$
        ]).pipe(
            map(([policies, search, filterId, group, lang]) => {
                if (policies.length === 0) {
                    if (search) {
                        return this.transloco.translate('policies.Policies.PoliciesList.Text.NoPoliciesOnSearch');
                    }
                    if (filterId && filterId !== 'all_policies')
                        return this.transloco.translate('policies.Policies.PoliciesList.Text.NoPolicies', {
                            filter: filterId
                        });
                    if (group) {
                        return this.transloco.translate('policies.Policies.PoliciesList.Text.NoPoliciesInGroup', {
                            group: group.info.name[lang as keyof ModelsGroupItemLocale]
                        });
                    }

                    return this.transloco.translate('policies.Policies.PoliciesList.Text.NoPoliciesInFilter');
                }

                return undefined;
            })
        );

        const gridFilterItemsSubscription: Subscription = this.policiesFacade.gridColumnFilterItems$.subscribe(
            (gridFilters) => {
                this.gridColumnsFilters = {
                    os: osList.map((os) => ({
                        ...os,
                        label: this.transloco.translate(os.label)
                    })),
                    ...gridFilters
                };
            }
        );
        this.subscription.add(gridFilterItemsSubscription);
    }

    private initPageStateRx() {
        // restore state
        combineLatest([this.policiesFacade.isInitialized$, this.sharedFacade.initializedGroups$])
            .pipe(
                skipWhile(
                    ([isPoliciesInitialized, isGroupsInitialized]) => !isPoliciesInitialized || !isGroupsInitialized
                ),
                take(1)
            )
            .subscribe(() => {
                this.policiesFacade.restoreState();
            });

        // save state
        const saveStateSubscription = combineLatest([
            this.policiesFacade.isRestored$,
            this.policiesFacade.selectedFilterId$,
            this.policiesFacade.selectedGroupId$,
            this.policiesFacade.gridFiltration$,
            this.policiesFacade.search$,
            this.policiesFacade.sorting$
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

    private initDataRx() {
        this.dataGrid$ = this.policies$.pipe(
            map((policies) => this.dateFormatterService.formatToRelativeDate(policies, 'created_date'))
        );
        this.policiesFacade.isRestored$
            .pipe(filter(Boolean), take(1))
            .subscribe(() => this.policiesFacade.fetchPoliciesPage());

        const entityForLinkingSubscription = this.policiesFacade.selectedPolicies$
            .pipe(map((policies) => policies[0]))
            .subscribe(this.facade.baseEntity$);
        this.subscription.add(entityForLinkingSubscription);

        const changeLinksSubscription = merge(
            this.policiesFacade.isLinkingPolicy$.pipe(pairwise()),
            this.policiesFacade.isUnlinkingPolicy$.pipe(pairwise())
        ).subscribe(([previous, current]) => {
            if (previous && !current) {
                this.policiesFacade.fetchPoliciesPage(1);
                this.policiesFacade.fetchFiltersCounters();
            }
        });
        this.subscription.add(changeLinksSubscription);
    }
}
