import {
    ChangeDetectionStrategy,
    Component,
    EventEmitter,
    Inject,
    Input,
    OnChanges,
    OnDestroy,
    OnInit,
    Output,
    SimpleChanges
} from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { McListSelectionChange } from '@ptsecurity/mosaic/list';
import {
    combineLatest,
    debounceTime,
    map,
    Observable,
    ReplaySubject,
    Subject,
    Subscription,
    withLatestFrom
} from 'rxjs';

import { PERMISSIONS_TOKEN } from '@soldr/core';
import { Group } from '@soldr/models';
import { SharedFacade } from '@soldr/store/shared';

import { Filter, FilterByGroup, LocaleItem, ProxyPermission, ViewMode } from '../../types';
import { DEBOUNCING_DURATION } from '../../utils';

@Component({
    selector: 'soldr-groups-and-filters',
    templateUrl: './groups-and-filters.component.html',
    styleUrls: ['./groups-and-filters.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class GroupsAndFiltersComponent implements OnInit, OnChanges, OnDestroy {
    @Input() filterId: string;
    @Input() filters: Filter[] = [];
    @Input() viewMode: ViewMode;

    @Output() selectFilter = new EventEmitter<string>();
    @Output() selectFilterByGroup = new EventEmitter<string>();

    currentSelectedFilterId: string[];
    filterId$ = new ReplaySubject<string>(1);
    filters$ = new ReplaySubject<Filter[]>(1);
    foundFilters$ = new Observable<Filter[]>();
    filtersByGroup$ = new Observable<FilterByGroup[]>();
    isLoadingGroups$: Observable<boolean>;
    language$: Observable<string>;
    searchValue$ = new ReplaySubject<string>(1);
    searchValue: string;
    subscription = new Subscription();
    themePalette = ThemePalette;
    selectedId$ = new Subject<string>();

    constructor(
        private transloco: TranslocoService,
        private sharedFacade: SharedFacade,
        @Inject(PERMISSIONS_TOKEN) public permitted: ProxyPermission
    ) {
        this.searchValue$.next('');
    }

    ngOnInit(): void {
        this.defineObservables();
    }

    ngOnChanges({ filters, filterId }: SimpleChanges): void {
        if (filters?.currentValue) {
            this.filters$.next(this.filters);
        }

        if (filterId?.currentValue) {
            this.filterId$.next(this.filterId);
        }
    }

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
    }

    defineObservables() {
        this.language$ = this.transloco.langChanges$.pipe(map((value: string) => value.split('-')[0]));
        this.isLoadingGroups$ = this.sharedFacade.isLoadingAllGroups$;

        // change selection on external change
        const filterIdSubscription = this.filterId$.subscribe((filterId) => {
            this.currentSelectedFilterId = [filterId];
        });

        // filters search
        const searchFilters = (filters: Filter[], searchValue: string) =>
            filters.filter((filter: Filter) =>
                filter.label.toLocaleLowerCase().includes(searchValue.toLocaleLowerCase())
            );
        this.foundFilters$ = combineLatest([this.filters$, this.searchValue$]).pipe(
            map(([filters, searchString]) => searchFilters(filters, searchString))
        );

        // groups search
        const searchValueSubscription = this.searchValue$
            .pipe(debounceTime(DEBOUNCING_DURATION))
            .subscribe((searchValue) => {
                this.sharedFacade.setSearchValue(searchValue);
                this.sharedFacade.fetchAllGroups();
            });

        const allGroupsBySearch$ = combineLatest([
            this.sharedFacade.allGroups$,
            this.sharedFacade.searchValue$,
            this.sharedFacade.selectedGroupTags$
        ]).pipe(
            map(([groups, search, tags]: [Group[], string, string[]]) =>
                !search && tags.length === 0
                    ? groups
                    : groups.filter(
                          (group: Group) =>
                              (group.info.name.ru.toLowerCase().includes(search.toLowerCase()) ||
                                  group.info.name.en.toLowerCase().includes(search.toLowerCase())) &&
                              (tags.length > 0 ? group.info.tags.some((tag) => tags.includes(tag)) : true)
                      )
            )
        );

        this.filtersByGroup$ = allGroupsBySearch$.pipe(
            map((groups: Group[]) =>
                groups.map(
                    (group) =>
                        ({
                            id: group.id.toString(),
                            hash: group.hash,
                            label: group.info.name as LocaleItem,
                            count:
                                this.viewMode === ViewMode.Agents
                                    ? group.details.agents
                                    : group.details?.policies?.length || 0
                        } as FilterByGroup)
                )
            )
        );

        // restore selection after search
        const restoreSelectedIdSubscription = combineLatest([this.filtersByGroup$, this.foundFilters$]).subscribe(
            () => {
                this.currentSelectedFilterId = [this.filterId];
            }
        );

        // emit events
        const selectedIdSubscription = this.selectedId$
            .pipe(withLatestFrom(this.filters$))
            .subscribe(([selectedId, filters]) => {
                const foundFilter = filters.find((item) => item.id === selectedId);

                if (foundFilter) {
                    this.selectFilter.emit(selectedId);
                } else {
                    this.selectFilterByGroup.emit(selectedId);
                }
            });

        this.subscription.add(restoreSelectedIdSubscription);
        this.subscription.add(selectedIdSubscription);
        this.subscription.add(searchValueSubscription);
        this.subscription.add(filterIdSubscription);
    }

    onSelectItem(change: McListSelectionChange) {
        this.selectedId$.next(change.option.value as string);
    }
}
