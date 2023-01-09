import {
    AfterContentInit,
    Component,
    ContentChildren,
    EventEmitter,
    Input,
    OnChanges,
    OnInit,
    Output,
    QueryList,
    SimpleChanges,
    ViewChild
} from '@angular/core';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { McInput } from '@ptsecurity/mosaic/input';
import { McPopoverComponent } from '@ptsecurity/mosaic/popover';
import {
    BehaviorSubject,
    combineLatest,
    combineLatestWith,
    filter,
    from,
    map,
    mergeWith,
    Observable,
    of,
    ReplaySubject,
    shareReplay,
    switchMap,
    toArray
} from 'rxjs';

import { ArrayFiltrationValue, SimpleFiltrationValue, sortGridColumnFilterItems, SortPipe } from '@soldr/shared';

import { Filtration, FiltrationValue } from '../../../types';
import { FilterItemComponent } from '../filter-item/filter-item.component';
import { GridColumnFilterItem } from '../grid.types';

@Component({
    selector: 'soldr-filter',
    templateUrl: './filter.component.html',
    styleUrls: ['./filter.component.scss'],
    providers: [SortPipe]
})
export class FilterComponent implements OnInit, OnChanges, AfterContentInit {
    /**
     * Поле для фильтрации. В случае его отсутствия - columnField.
     */
    @Input() field: string;
    /**
     * Заголовок фильтра, по умолчанию берется из колонки, указанной columnField
     */
    @Input() title: string;
    /**
     * Placeholder, по умолчанию берется из колонки, указанной columnField
     */
    @Input() placeholder: string;
    /**
     * Тип выбираемого значения, одиночное и множественные
     */
    @Input() multiple: boolean;
    /**
     * Предустановленные значения
     */
    @Input() selectedValues: FiltrationValue;

    @Output() changeValue = new EventEmitter<Filtration>();

    @ViewChild('filterPopover') filterPopover: McPopoverComponent;
    @ViewChild('searchInput', { read: McInput }) searchInput: McInput;

    @ContentChildren(FilterItemComponent) filterItemsComponents: QueryList<FilterItemComponent>;

    filterItemSearch = new ReplaySubject<string>(1);
    filterItemSearch$ = this.filterItemSearch.asObservable().pipe(shareReplay({ bufferSize: 1, refCount: true }));
    filterItems$: Observable<FilterItemComponent[]>;
    foundFilterItems$: Observable<GridColumnFilterItem[]>;
    labels$: Observable<string[]>;
    searchValue = '';
    selectedFilterValues$ = new BehaviorSubject<ArrayFiltrationValue>([]);
    normalizedSelectedFilterValues$ = this.selectedFilterValues$.pipe(
        map((values) => (values || []).map((v: SimpleFiltrationValue) => FilterComponent.getNormalizedValue(v)))
    );
    themePalette = ThemePalette;

    constructor(private sortPipe: SortPipe) {}

    private static getNormalizedValue(value: SimpleFiltrationValue) {
        return typeof value === 'string' ? value.toLowerCase() : value;
    }

    ngAfterContentInit() {
        this.filterItems$ = of(this.filterItemsComponents.toArray()).pipe(
            mergeWith(
                this.filterItemsComponents.changes.pipe(
                    shareReplay({ bufferSize: 1, refCount: true }),
                    map((v) => v.toArray())
                )
            )
        );

        this.foundFilterItems$ = combineLatest([this.filterItemSearch$, this.filterItems$]).pipe(
            switchMap(([searchValue, filterItems]) =>
                from(filterItems.map((item: FilterItemComponent) => ({ label: item.label, value: item.value }))).pipe(
                    filter((item: GridColumnFilterItem) =>
                        item.label?.toLocaleLowerCase().includes(searchValue?.toLocaleLowerCase())
                    ),
                    toArray()
                )
            ),
            combineLatestWith(this.normalizedSelectedFilterValues$),
            map(([items, selected]) =>
                this.sortPipe
                    .transform(items, sortGridColumnFilterItems)
                    .sort(this.sortByEnabledState(selected as string[]))
            )
        );

        this.labels$ = combineLatest([this.normalizedSelectedFilterValues$, this.filterItems$]).pipe(
            map(([selectedValues, filterItems]) =>
                (selectedValues?.length
                    ? filterItems?.filter(({ value }: { value: SimpleFiltrationValue }) =>
                          Array.isArray(selectedValues)
                              ? selectedValues.includes(FilterComponent.getNormalizedValue(value))
                              : selectedValues === value
                      ) || []
                    : []
                ).map(({ label }) => label)
            )
        );
    }

    ngOnInit(): void {
        this.filterItemSearch.next('');
    }

    ngOnChanges({ title, selectedValues }: SimpleChanges): void {
        if (title?.currentValue && !this.placeholder) {
            this.placeholder = this.title;
        }

        if (selectedValues) {
            this.selectedFilterValues$.next([
                ...((this.selectedValues as ArrayFiltrationValue) || []).map((v: SimpleFiltrationValue) =>
                    FilterComponent.getNormalizedValue(v)
                )
            ]);
        }
    }

    getSelectedValues(selectedFilterValues: ArrayFiltrationValue, value: SimpleFiltrationValue) {
        return selectedFilterValues?.includes(FilterComponent.getNormalizedValue(value));
    }

    popoverVisibleChange(value: boolean) {
        if (!value) {
            this.searchValue = '';
            this.filterItemSearch.next('');
        } else {
            this.searchInput?.focus();
        }
        this.selectedFilterValues$.next([...((this.selectedValues as string[]) || [])]);
    }

    onChangeSelectedFilterItems(currentValues: ArrayFiltrationValue, value: string, isSelected: boolean) {
        const values = currentValues || [];
        if (isSelected) {
            this.selectedFilterValues$.next([...values, value]);
        } else {
            this.selectedFilterValues$.next(values.filter((filterValue) => filterValue !== value));
        }
    }

    setFiltration(value: FiltrationValue) {
        this.changeValue.emit({ field: this.field, value: [value] } as Filtration);
    }

    apply(selectedFilterValues: ArrayFiltrationValue) {
        this.changeValue.emit({ field: this.field, value: selectedFilterValues });
        this.filterPopover.hide(0);
    }

    cancel() {
        this.selectedFilterValues$.next([...((this.selectedValues as string[]) || [])]);
        this.filterPopover.hide(0);
    }

    reset() {
        this.selectedFilterValues$.next([]);
    }

    private sortByEnabledState(selectedValues: string[]) {
        return (a: GridColumnFilterItem, b: GridColumnFilterItem) => {
            const isSelectedA = Number(selectedValues.includes(a.value as string));
            const isSelectedB = Number(selectedValues.includes(b.value as string));

            return isSelectedB - isSelectedA;
        };
    }
}
