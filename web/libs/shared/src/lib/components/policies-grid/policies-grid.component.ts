import { Component, EventEmitter, Input, OnChanges, OnDestroy, Output, SimpleChanges } from '@angular/core';
import { PopUpPlacements } from '@ptsecurity/mosaic/core';
import { BehaviorSubject, Subscription } from 'rxjs';

import { GroupModule, Policy } from '@soldr/models';
import {
    Filtration,
    LanguageService,
    osList,
    sortModules,
    sortTags,
    sortGroups,
    GridColumnFilterItem,
    Sorting,
    ViewMode
} from '@soldr/shared';

@Component({
    selector: 'soldr-policies-grid',
    templateUrl: './policies-grid.component.html',
    styleUrls: ['./policies-grid.component.scss']
})
export class PoliciesGridComponent implements OnChanges, OnDestroy {
    @Input() policies: Policy[];
    @Input() gridFilters: { [field: string]: GridColumnFilterItem[] };
    @Input() gridFiltrationByFields: Record<string, Filtration>;
    @Input() hideGroupsColumn: boolean;
    @Input() isLoading: boolean;
    @Input() modules: GroupModule[];
    @Input() searchValue: string;
    @Input() selectedPolicy: Policy;
    @Input() storageGridKey: string;
    @Input() stateStorageKey: string;
    @Input() stateStorageValue: any;
    @Input() total: number;
    @Input() viewMode: ViewMode;

    @Output() changeSelection = new EventEmitter<Policy>();
    @Output() filter = new EventEmitter<Filtration>();
    @Output() loadNextPage = new EventEmitter();
    @Output() search = new EventEmitter<string>();
    @Output() refresh = new EventEmitter();
    @Output() resetFiltration = new EventEmitter();
    @Output() setTag = new EventEmitter<string>();
    @Output() gridSorting = new EventEmitter<Sorting>();
    @Output() sort = new EventEmitter<Sorting>();

    policies$ = new BehaviorSubject<Policy[]>([]);
    language$ = this.languageService.current$;
    search$ = new BehaviorSubject<string>('');

    placement = PopUpPlacements;
    sortModules = sortModules;
    sortTags = sortTags;
    sortGroups = sortGroups;
    gridFiltration: Filtration[] = [];
    subscription = new Subscription();
    gridColumnsFilters: { [field: string]: GridColumnFilterItem[] } = {
        modules: [],
        tags: [],
        groups: [],
        os: [...osList]
    };

    constructor(private languageService: LanguageService) {}

    ngOnChanges({ policies, searchValue, gridFilters, gridFiltrationByFields }: SimpleChanges): void {
        if (policies?.currentValue) {
            this.policies$.next(this.policies);
        }

        if (searchValue) {
            this.search$.next(this.searchValue || '');
        }

        if (gridFilters?.currentValue) {
            this.gridColumnsFilters = {
                os: [...this.gridColumnsFilters.os],
                ...this.gridFilters
            };
        }

        if (gridFiltrationByFields?.currentValue) {
            this.gridFiltration = Object.values(this.gridFiltrationByFields);
        }
    }

    ngOnDestroy() {
        this.subscription.unsubscribe();
    }

    onSearch(value: string) {
        this.search.next(value);
    }

    nextPage() {
        this.loadNextPage.emit();
    }

    onGridSelectRows([policy]: Policy[]) {
        this.changeSelection.emit(policy);
    }

    onResetFiltration() {
        this.resetFiltration.emit();
    }

    onSetTag(tag: string) {
        this.setTag.emit(tag);
    }

    onGridFilter(filtration: Filtration) {
        this.filter.next(filtration);
    }

    onGridSort(sorting: Sorting): void {
        this.sort.emit(sorting);
    }
}
