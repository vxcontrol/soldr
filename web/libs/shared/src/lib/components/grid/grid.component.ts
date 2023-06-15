import { CdkScrollable, ScrollDispatcher } from '@angular/cdk/overlay';
import {
    Component,
    ContentChild,
    ContentChildren,
    ElementRef,
    EventEmitter,
    Input,
    NgZone,
    OnChanges,
    OnDestroy,
    OnInit,
    Output,
    QueryList,
    SimpleChanges,
    TemplateRef,
    ViewChild,
    ViewEncapsulation
} from '@angular/core';
import { defaultRefresherOptions } from '@mosaic-design/infosec-components/components/refresher';
import { TranslocoService } from '@ngneat/transloco';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { McListSelectionChange } from '@ptsecurity/mosaic/list';
import { McPopoverComponent } from '@ptsecurity/mosaic/popover';
import { AgGridEvent, ColDef, ColumnApi, GridApi, GridOptions, ICellRendererParams } from 'ag-grid-community';
import { GetRowIdParams } from 'ag-grid-community/dist/lib/entities/iCallbackParams';
import {
    BehaviorSubject,
    combineLatest,
    filter,
    from,
    map,
    mergeMap,
    Observable,
    of,
    shareReplay,
    Subject,
    Subscription,
    switchMap,
    toArray
} from 'rxjs';

import { Filtration } from '../../types';

import { GridActionBarDirective } from './actionbar/grid-action-bar.directive';
import { ColumnComponent } from './column/column.component';
import { FilterComponent } from './filter/filter.component';
import { GridFooterDirective } from './footer/grid-footer.directive';
import { GridColumnDef, GridsState, LocalizedData, Selection, Sorting, SortingDirection } from './grid.types';
import { NoRowsOverlayComponent } from './no-rows-overlay/no-rows-overlay.component';
import { TemplateCellComponent } from './template-cell/template-cell.component';

const AG_GRID_FOCUSED_CLASS = 'ag-grid-focused';

@Component({
    selector: 'soldr-grid',
    templateUrl: './grid.component.html',
    styleUrls: ['./grid.component.scss'],
    encapsulation: ViewEncapsulation.None
})
export class GridComponent implements OnInit, OnChanges, OnDestroy {
    @Input() autoHeight: boolean;
    @Input() data: any[] = [];
    @Input() emptyText: string;
    @Input() exportAllTemplate: TemplateRef<any>;
    @Input() exportSelectedTemplate: TemplateRef<any>;
    @Input() filtration: Filtration[];
    @Input() footerTemplate: TemplateRef<any>;
    @Input() hasHeader = true;
    @Input() identityField = 'id';
    @Input() isLoading: boolean;
    @Input() noRowsOverlayTemplate: TemplateRef<any>;
    @Input() placeholder = '';
    @Input() rowHeight: number;
    @Input() searchString: string;
    @Input() selected: any[] = [];
    @Input() selectionType: Selection = Selection.Single;
    @Input() sorting: Sorting | Record<never, any>;
    @Input() storageKey: string;
    @Input() total = 0;

    @Output() addedNewRows = new EventEmitter();
    @Output() export = new EventEmitter<{ selected?: any[]; columns: string[] }>();
    @Output() nextPage = new EventEmitter();
    @Output() refresh = new EventEmitter();
    @Output() resetFiltration = new EventEmitter();
    @Output() search = new EventEmitter<string>();
    @Output() selectRows = new EventEmitter();
    @Output() sortChanged = new EventEmitter<Sorting | Record<never, any>>();

    @ViewChild('columnsStatePopover', { static: false }) columnsStatePopover: McPopoverComponent;
    @ContentChildren(ColumnComponent) columns: QueryList<ColumnComponent>;
    @ContentChildren(FilterComponent) filters: QueryList<FilterComponent>;
    @ContentChild(GridActionBarDirective) gridActionBar: any;
    @ContentChild(GridFooterDirective) gridFooter: any;

    canShowFiltersBlock = false;
    columnsDefs$ = new BehaviorSubject<GridColumnDef[]>([]);
    columnsDefs: GridColumnDef[];
    columnsSearch = new Subject<string>();
    columnsSearch$ = this.columnsSearch.asObservable().pipe(shareReplay({ bufferSize: 1, refCount: true }));
    foundColumns$: Observable<GridColumnDef[]>;
    components = {
        cellRenderer: TemplateCellComponent,
        noRowsComponent: NoRowsOverlayComponent
    };
    gridOptions: GridOptions = {
        headerHeight: 32,
        rowHeight: 40,
        suppressMultiSort: true,
        suppressFocusAfterRefresh: true,
        defaultColDef: {
            minWidth: 32,
            comparator: () => 0
        },
        getRowId: (params: GetRowIdParams) => params.data[this.identityField],
        noRowsOverlayComponent: 'noRowsComponent'
    };
    scrollableViewPort: CdkScrollable;
    searchValue: string;
    selectedVisibleColumns: string[] = [];
    selection = Selection;
    themePalette = ThemePalette;

    get domLayout() {
        return this.autoHeight ? 'autoHeight' : 'normal';
    }

    public gridApi: GridApi;

    private columnApi: ColumnApi;
    private subscription: Subscription = new Subscription();
    private gridsStorageKey = 'grids';

    private static getColDefByColumn(column: ColumnComponent): GridColumnDef {
        return {
            autoHeight: column.autoHeight,
            autoSize: column.autoSize,
            cellClass: column.cellClass,
            cellRenderer: column.template ? 'cellRenderer' : (params: ICellRendererParams) => params.value,
            cellRendererParams: {
                template: column.template
            },
            cellStyle: column.cellStyle,
            colId: column.field,
            default: column.default,
            field: column.field,
            filtrationField: column.filtrationField,
            flex: column.flex,
            headerComponentParams: { displayName: column.displayName },
            headerName: column.headerName,
            maxWidth: column.maxWidth,
            minWidth: column.minWidth,
            pinned: column.pinned,
            required: column.required,
            resizable: column.resizable,
            sortField: column.sortField,
            sortable: column.sortable,
            width: column.width,
            wrapText: column.wrapText
        };
    }

    constructor(
        private hostAsElement: ElementRef,
        private transloco: TranslocoService,
        private scrollDispatcher: ScrollDispatcher,
        private element: ElementRef,
        private zone: NgZone
    ) {}

    private restoreSorting() {
        const sortingParams = this.sorting as Sorting;

        if (!sortingParams?.prop) {
            return;
        }

        this.columnApi?.applyColumnState({
            state: [
                {
                    colId:
                        this.columnsDefs.find(({ sortField }) => sortField === sortingParams.prop)?.colId ||
                        sortingParams.prop,
                    sort:
                        sortingParams?.order === SortingDirection.ASC
                            ? 'asc'
                            : sortingParams?.order === SortingDirection.DESC
                            ? 'desc'
                            : undefined
                }
            ]
        });
    }

    ngOnInit(): void {
        this.foundColumns$ = combineLatest([this.columnsDefs$, this.columnsSearch$]).pipe(
            switchMap(([columnDefs, searchValue]) =>
                from(columnDefs).pipe(
                    mergeMap((item) =>
                        of(item.headerName).pipe(
                            map((headerName) => ({
                                origin: item,
                                localizedData: { headerName }
                            }))
                        )
                    ),
                    filter((item: LocalizedData<GridColumnDef>) =>
                        item.localizedData.headerName?.toLocaleLowerCase().includes(searchValue?.toLocaleLowerCase())
                    ),
                    map((item: LocalizedData<GridColumnDef>) => item.origin),
                    toArray(),
                    map((items) => items.sort((a, b) => (a.headerName > b.headerName ? 1 : -1)))
                )
            )
        );

        const columnsSubscription = this.columnsDefs$.subscribe((defs) => {
            this.columnsDefs = defs;

            this.gridApi?.setColumnDefs(defs);
            this.gridApi?.refreshHeader();
        });

        this.subscription.add(columnsSubscription);
    }

    ngOnChanges({
        filtration,
        sorting,
        searchString,
        hasHeader,
        rowHeight,
        data,
        total,
        emptyText,
        isLoading,
        noRowsOverlayTemplate
    }: SimpleChanges): void {
        if (sorting?.currentValue) {
            this.restoreSorting();
        }

        if (searchString?.currentValue || searchString?.currentValue === '') {
            this.searchValue = this.searchString;
        }

        if (hasHeader) {
            this.gridOptions = {
                ...this.gridOptions,
                headerHeight: this.hasHeader ? this.gridOptions.headerHeight : 0
            };
        }

        if (rowHeight?.currentValue) {
            this.gridOptions = {
                ...this.gridOptions,
                rowHeight: this.rowHeight
            };
        }

        if (filtration?.currentValue) {
            this.displayColumnsFromFiltration();
        }

        if (
            (data?.currentValue?.length || total?.currentValue) &&
            total?.currentValue > total?.previousValue &&
            data?.previousValue?.length &&
            this.gridApi
        ) {
            this.addedNewRows.emit();
        }

        if (noRowsOverlayTemplate?.currentValue || emptyText?.currentValue || isLoading) {
            this.gridOptions.noRowsOverlayComponentParams = {
                emptyText: this.emptyText,
                isFirstLoading: isLoading?.firstChange,
                template: this.noRowsOverlayTemplate
            };
        }
    }

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
        this.deregisterViewportAsScrollable();
    }

    onFocusOut(): void {
        this.hostAsElement.nativeElement.classList.remove(AG_GRID_FOCUSED_CLASS);
    }

    onFocusIn(): void {
        this.hostAsElement.nativeElement.classList.add(AG_GRID_FOCUSED_CLASS);
    }

    onKeyUp({ ctrlKey, code }: KeyboardEvent): boolean | void {
        if (this.selectionType === Selection.Multiple && ctrlKey && code === 'KeyA') {
            this.gridApi.selectAll();

            return false;
        }
    }

    resetFilters() {
        this.resetFiltration.emit();
    }

    toggleFiltersBlock() {
        this.canShowFiltersBlock = !this.canShowFiltersBlock;
    }

    onOpenColumnsPopover() {
        this.selectedVisibleColumns = this.columnApi.getAllDisplayedColumns().map((column) => column.getColDef().field);
        this.columnsSearch.next('');
    }

    onCloseColumnsPopover() {
        this.columnsSearch.next('');
    }

    columnsPopoverVisibleChange(value: boolean) {
        if (value) {
            this.onOpenColumnsPopover();
        } else {
            this.onCloseColumnsPopover();
        }
    }

    onChangeColumnsVisibility($event: McListSelectionChange) {
        if ($event.option.selected) {
            this.selectedVisibleColumns.push($event.option.value as string);
        } else {
            this.selectedVisibleColumns = this.selectedVisibleColumns.filter(
                (column) => column !== $event.option.value
            );
        }
    }

    gridReady($event: AgGridEvent) {
        this.gridApi = $event.api;
        this.columnApi = $event.columnApi;
        const defs: ColDef[] = this.columns.map((column) => GridComponent.getColDefByColumn(column));
        this.columnsDefs$.next(defs);

        this.restoreState();
        this.restoreSorting();

        this.registerViewportAsScrollable();
    }

    rowDataChanged() {
        for (const node of this.gridApi?.getSelectedNodes() || []) {
            const colId = this.gridApi.getFocusedCell().column.getColId();
            const row = this.gridApi?.getRowNode(node.id);

            row.selectThisNode(true);
            this.gridApi?.setFocusedCell(row.rowIndex, colId);
        }

        const selectedNodes = this.gridApi?.getSelectedNodes();

        this.gridApi?.refreshCells({ force: true });

        if (selectedNodes?.length === 0) {
            const defaultSelectedNode = (this.gridApi?.getRenderedNodes() || [])[0];

            if (defaultSelectedNode) {
                defaultSelectedNode.setSelected(true, true);
                this.gridApi.setFocusedCell(0, (this.gridApi.getColumnDefs()[0] as ColDef).colId);
            }
        }
    }

    sortChangedCallback() {
        const sortData = this.columnApi
            .getColumnState()
            .map((item) => ({
                prop: this.columnsDefs.find((def) => def.colId === item.colId)?.sortField || item.colId,
                order:
                    item.sort === 'asc'
                        ? SortingDirection.ASC
                        : item?.sort === 'desc'
                        ? SortingDirection.DESC
                        : undefined
            }))
            .filter((item) => !!item.order);

        if (sortData.length === 0) {
            this.sortChanged.emit({});

            return;
        }

        this.sortChanged.emit(sortData[0]);
    }

    nextPageCallback() {
        this.nextPage.emit();
    }

    resetColumnsState() {
        this.selectedVisibleColumns = this.columnsDefs
            .filter((columnDef) => columnDef.default || columnDef.required)
            .map((columnDef) => columnDef.field);

        if (this.selectedVisibleColumns.length === 0) {
            this.selectedVisibleColumns = this.columnsDefs.map((columnDef) => columnDef.field);
        }
    }

    applyColumnsState() {
        this.columnApi.setColumnsVisible(this.columnApi.getAllDisplayedColumns(), false);
        this.columnApi.setColumnsVisible(this.selectedVisibleColumns, true);
        this.columnsStatePopover?.hide(0);
    }

    cancelColumnsState() {
        this.selectedVisibleColumns = [];
        this.columnsStatePopover.hide(0);
    }

    onChangeSearch() {
        this.search.emit(this.searchValue);
    }

    clearSearch() {
        this.search.emit(undefined);
    }

    onSelectionChange($event: AgGridEvent) {
        const selectedRows = $event.api.getSelectedRows();
        this.selectRows.emit(selectedRows);
    }

    saveState() {
        const gridsState: GridsState = (JSON.parse(window.localStorage.getItem(this.gridsStorageKey)) ||
            {}) as GridsState;

        gridsState[this.storageKey] = this.columnApi.getColumnState();
        window.localStorage.setItem(this.gridsStorageKey, JSON.stringify(gridsState));
    }

    restoreState() {
        const savedState = this.getSavedState();

        savedState.forEach((col) => (col.sort = undefined));

        if (savedState.length > 0) {
            this.displayColumnsFromFiltration();
            this.columnApi?.applyColumnState({ state: savedState, applyOrder: true });
        } else {
            this.resetColumnsState();
            this.applyColumnsState();
        }
    }

    onFirstDataRendered() {
        const savedState = this.getSavedState();

        const columnsWithAutoSize = this.columnsDefs
            .filter(({ autoSize, field }) => {
                const hasState = !!savedState.find(({ colId }) => colId === field);

                return !hasState && autoSize;
            })
            .map(({ field }) => field);
        this.columnApi.autoSizeColumns(columnsWithAutoSize, true);
    }

    onExportSelected() {
        this.export.emit({ selected: this.selected, columns: this.displayedColumnsIds });
    }

    onExportAll() {
        this.export.emit({ columns: this.displayedColumnsIds });
    }

    get displayedColumnsIds() {
        return this.columnApi.getAllDisplayedColumns().map((column) => column.getColId());
    }

    get selectedRows() {
        return this.gridApi?.getSelectedRows().length || 0;
    }

    private getSavedState() {
        const gridsState: GridsState = (JSON.parse(window.localStorage.getItem(this.gridsStorageKey)) ||
            {}) as GridsState;

        return gridsState[this.storageKey] || [];
    }

    private registerViewportAsScrollable() {
        const bodyViewportRef = new ElementRef(
            this.element.nativeElement.querySelector('.ag-body-viewport')
        ) as ElementRef<HTMLElement>;
        this.scrollableViewPort = new CdkScrollable(bodyViewportRef, this.scrollDispatcher, this.zone);
        this.scrollDispatcher.register(this.scrollableViewPort);
    }

    private deregisterViewportAsScrollable() {
        this.scrollDispatcher.deregister(this.scrollableViewPort);
    }

    private displayColumnsFromFiltration() {
        const filtrationColumns = this.filtration
            ?.map(
                (filtrationItem) =>
                    this.columnsDefs?.find(({ field, filtrationField }) =>
                        [field, filtrationField].includes(filtrationItem.field)
                    )?.colId
            )
            .filter((value) => !!value);

        this.columnApi?.setColumnsVisible(filtrationColumns || [], true);
    }
}
