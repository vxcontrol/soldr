import { ChangeDetectionStrategy, Component, Inject, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { ActivatedRoute, Params, Router } from '@angular/router';
import { TranslocoService } from '@ngneat/transloco';
import { PopUpPlacements } from '@ptsecurity/mosaic/core';
import { SidebarPositions } from '@ptsecurity/mosaic/sidebar';
import { Direction } from '@ptsecurity/mosaic/splitter';
import { combineLatest, filter, first, map, merge, Observable, pairwise, skipWhile, Subscription, take } from 'rxjs';

import { PERMISSIONS_TOKEN } from '@soldr/core';
import { Group, Policy } from '@soldr/models';
import {
    Filtration,
    GridColumnFilterItem,
    LanguageService,
    Sorting,
    STATE_STORAGE_TOKEN,
    StateStorage,
    WidthChangeEvent,
    sortModules,
    sortPolicies,
    sortTags,
    PageTitleService,
    DateFormatterService,
    mergeDeep,
    POLICY_LINKING_FACADE,
    LinkPolicyToGroupFacade,
    ProxyPermission,
    ViewMode,
    GridComponent
} from '@soldr/shared';
import { GroupsFacade } from '@soldr/store/groups';
import { SharedFacade } from '@soldr/store/shared';

import { GroupsExporterService } from '../../services';
import { defaultGroupsPageState, GroupsPageState } from '../../utils';

@Component({
    selector: 'soldr-groups-page',
    templateUrl: './groups-page.component.html',
    styleUrls: ['./groups-page.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class GroupsPageComponent implements OnInit, OnDestroy {
    direction = Direction;
    emptyText$: Observable<string>;
    dataGrid$: Observable<Group[]>;
    gridColumnsFilters: { [field: string]: GridColumnFilterItem[] } = { modules: [], policies: [], tags: [] };
    gridFiltration$ = this.groupsFacade.gridFiltration$;
    gridFiltrationByField$ = this.groupsFacade.gridFiltrationByField$;
    groups$ = this.groupsFacade.groups$;
    group$ = this.groupsFacade.group$;
    groupModules$ = this.groupsFacade.groupModules$;
    isCopyingGroup$ = this.groupsFacade.isCopyingGroup$;
    isCreatingGroup$ = this.groupsFacade.isCreatingGroup$;
    isDeletingGroup$ = this.groupsFacade.isDeletingGroup$;
    isLinkingGroup$ = this.groupsFacade.isLinkingGroup$;
    isLoading$ = this.groupsFacade.isLoadingGroups$;
    isLoadingGroup$ = this.groupsFacade.isLoadingGroup$;
    isUpdatingGroup$ = this.groupsFacade.isUpdatingGroup$;
    language$ = this.languageService.current$;
    page$ = this.groupsFacade.page$;
    pageState: GroupsPageState;
    placement = PopUpPlacements;
    searchString$ = this.groupsFacade.search$;
    selected$ = this.groupsFacade.selectedGroups$;
    sidebarPositions = SidebarPositions;
    sortModules = sortModules;
    sortPolicies = sortPolicies;
    sortTags = sortTags;
    sorting$ = this.groupsFacade.sorting$;
    subscription = new Subscription();
    total$ = this.groupsFacade.total$;
    viewModeEnum = ViewMode;

    constructor(
        private router: Router,
        private activatedRoute: ActivatedRoute,
        private dateFormatterService: DateFormatterService,
        private groupsExporter: GroupsExporterService,
        private languageService: LanguageService,
        private transloco: TranslocoService,
        private groupsFacade: GroupsFacade,
        private pageTitleService: PageTitleService,
        private sharedFacade: SharedFacade,
        @Inject(PERMISSIONS_TOKEN) public permitted: ProxyPermission,
        @Inject(STATE_STORAGE_TOKEN) private stateStorage: StateStorage,
        @Inject(POLICY_LINKING_FACADE) private facade: LinkPolicyToGroupFacade<Policy, Group>
    ) {
        this.sharedFacade.fetchAllModules();
        this.sharedFacade.fetchAllPolicies();
    }

    ngOnInit(): void {
        this.defineTitle();
        this.initPageStateRx();
        this.initDataRx();
        this.initGridRx();

        this.pageState = mergeDeep(
            defaultGroupsPageState(),
            (this.stateStorage.loadState('groups.list') as GroupsPageState) || {}
        );
    }

    ngOnDestroy(): void {
        this.groupsFacade.reset();
        this.subscription.unsubscribe();
    }

    onGridFilter(filtration: Filtration) {
        this.groupsFacade.setGridFiltration(filtration);
    }

    onGridSearch(value: string) {
        this.groupsFacade.setGridSearch(value);
    }

    onGridSort(sorting: Sorting) {
        this.groupsFacade.setGridSorting(sorting);
    }

    onGridSelectRows(groups: Group[]) {
        this.groupsFacade.resetGroupErrors();
        this.groupsFacade.selectGroups(groups);
    }

    onResetFiltration() {
        this.groupsFacade.resetFiltration();
    }

    setTag(tag: string) {
        this.groupsFacade.setGridFiltrationByTag(tag);
    }

    loadNextPage(page: number) {
        this.groupsFacade.fetchGroupsPage(page);
    }

    refreshSelected() {
        this.groupsFacade.createdGroup$.pipe(filter(Boolean), first()).subscribe(({ hash }) =>
            setTimeout(() => {
                this.router.navigate(['/groups', hash]);
                this.groupsFacade.resetCreatedGroup();
            })
        );
    }

    saveRightSidebarState(opened: boolean) {
        this.pageState.rightSidebar = { ...this.pageState.rightSidebar, opened };
    }

    saveRightSidebarWidth($event: WidthChangeEvent) {
        if ($event.width !== '32px') {
            this.pageState.rightSidebar = { ...this.pageState.rightSidebar, width: $event.width };
        }
    }

    refreshData() {
        this.groupsFacade.fetchGroupsPage(1);
    }

    onExport($event: { selected?: any[]; columns: string[] }) {
        this.groupsExporter.export($event.columns, $event.selected as Group[]);
    }

    private defineTitle() {
        const titlesSubscription = combineLatest([
            this.transloco.selectTranslate('Groups.PageTitle.Text.Groups', {}, 'groups'),
            this.sharedFacade.selectedServiceName$,
            this.transloco.selectTranslate('Shared.Pseudo.PageTitle.ApplicationName', {}, 'shared')
        ])
            .pipe(map((segments) => segments.filter(Boolean)))
            .subscribe((segments) => this.pageTitleService.setTitle(segments));

        this.subscription.add(titlesSubscription);
    }

    private initDataRx() {
        this.dataGrid$ = this.groups$.pipe(
            map((group) => this.dateFormatterService.formatToRelativeDate(group, 'created_date'))
        );
        this.groupsFacade.isRestored$.pipe(filter(Boolean), take(1)).subscribe(() => {
            this.groupsFacade.fetchGroupsPage(1);
        });

        const entityForLinkingSubscription = this.groupsFacade.selectedGroups$
            .pipe(map((groups) => groups[0]))
            .subscribe(this.facade.baseEntity$);
        this.subscription.add(entityForLinkingSubscription);

        const changeLinksSubscription = merge(
            this.groupsFacade.isLinkingGroup$.pipe(pairwise()),
            this.groupsFacade.isUnlinkingGroup$.pipe(pairwise())
        ).subscribe(([previous, current]) => {
            if (previous && !current) {
                this.groupsFacade.fetchGroupsPage(1);
            }
        });
        this.subscription.add(changeLinksSubscription);
    }

    private initGridRx() {
        this.emptyText$ = combineLatest([this.groupsFacade.groups$, this.groupsFacade.search$]).pipe(
            map(([groups, search]) => {
                if (groups.length === 0) {
                    if (search) return this.transloco.translate('groups.Groups.GroupsList.Text.NoGroupsOnSearch');

                    return this.transloco.translate('groups.Groups.GroupsList.Text.NoGroups');
                }

                return undefined;
            })
        );

        const gridFilterItemsSubscription: Subscription = this.groupsFacade.gridColumnFilterItems$.subscribe(
            (gridFilters) => {
                this.gridColumnsFilters = { ...gridFilters };
            }
        );
        this.subscription.add(gridFilterItemsSubscription);
    }

    private initPageStateRx() {
        // restore state
        this.groupsFacade.restoreState();

        // save state
        const saveStateSubscription = combineLatest([
            this.groupsFacade.isRestored$,
            this.groupsFacade.gridFiltration$,
            this.groupsFacade.search$,
            this.groupsFacade.sorting$
        ])
            .pipe(skipWhile(([restored]) => !restored))
            .subscribe(([, filtration, search, sorting]) => {
                const queryParams: Params = {
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
}
