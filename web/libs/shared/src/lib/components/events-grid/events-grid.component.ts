import {
    Component,
    EventEmitter,
    Inject,
    Input,
    OnChanges,
    OnDestroy,
    Output,
    SimpleChanges,
    TemplateRef,
    ViewChild
} from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';
import { Direction } from '@ptsecurity/mosaic/splitter';
import { BehaviorSubject, Subscription } from 'rxjs';

import { ModelsEvent } from '@soldr/api';
import { PERMISSIONS_TOKEN } from '@soldr/core';
import { Event } from '@soldr/models';

import { STATE_STORAGE_TOKEN, StateStorage, WidthChangeEvent } from '../../directives';
import { LanguageService } from '../../services';
import { Filtration, ProxyPermission, ViewMode } from '../../types';
import { GridColumnFilterItem, Sorting } from '../grid/grid.types';

interface EventsGridState {
    rightSidebar: {
        width: string;
    };
}

const defaultState = (): EventsGridState => ({
    rightSidebar: {
        width: '320px'
    }
});

@Component({
    selector: 'soldr-events-grid',
    templateUrl: './events-grid.component.html',
    styleUrls: ['./events-grid.component.scss']
})
export class EventsGridComponent implements OnChanges, OnDestroy {
    @Input() events: Event[];
    @Input() gridFilters: { [field: string]: GridColumnFilterItem[] };
    @Input() gridFiltrationByFields: Record<string, Filtration>;
    @Input() hideModuleColumn: boolean;
    @Input() isLoading: boolean;
    @Input() moduleLink: TemplateRef<any>;
    @Input() searchValue: string;
    @Input() storageKey: string;
    @Input() total: number;
    @Input() viewMode: ViewMode;

    @Output() filter = new EventEmitter<Filtration>();
    @Output() loadNextPage = new EventEmitter();
    @Output() refresh = new EventEmitter();
    @Output() resetFiltration = new EventEmitter();
    @Output() search = new EventEmitter<string>();
    @Output() sort = new EventEmitter<Sorting>();

    @ViewChild('eventSidebar') eventSidebar: TemplateRef<any>;

    direction = Direction;
    event: Event;
    events$ = new BehaviorSubject<ModelsEvent[]>([]);
    gridColumnsFilters: { [field: string]: GridColumnFilterItem[] } = {
        modules: [],
        agents: [],
        groups: [],
        policies: []
    };
    gridFiltration: Filtration[] = [];
    language$ = this.languageService.current$;
    search$ = new BehaviorSubject<string>('');
    subscription = new Subscription();
    viewModeEnum = ViewMode;
    state: EventsGridState;

    constructor(
        private languageService: LanguageService,
        private transloco: TranslocoService,
        @Inject(STATE_STORAGE_TOKEN) private stateStorage: StateStorage,
        @Inject(PERMISSIONS_TOKEN) public permitted: ProxyPermission
    ) {
        this.state = {
            ...defaultState(),
            ...((this.stateStorage.loadState('events.list') as EventsGridState) || {})
        };
    }

    ngOnChanges({ events, searchValue, gridFilters, gridFiltrationByFields }: SimpleChanges): void {
        if (events?.currentValue) {
            this.events$.next(this.events);
        }

        if (searchValue) {
            this.search$.next(this.searchValue || '');
        }

        if (gridFilters?.currentValue) {
            this.gridColumnsFilters = { ...this.gridFilters };
        }

        if (gridFiltrationByFields?.currentValue) {
            this.gridFiltration = Object.values(this.gridFiltrationByFields);
        }
    }

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
    }

    nextPage() {
        this.loadNextPage.emit();
    }

    onSearch(value: string) {
        this.search.next(value);
    }

    onGridSelectRows([event]: Event[]) {
        this.event = event;
    }

    onGridFilter(filtration: Filtration) {
        this.filter.next(filtration);
    }

    onResetFiltration() {
        this.resetFiltration.emit();
    }

    get placeholderKey() {
        switch (this.viewMode) {
            case ViewMode.Policies:
                return this.transloco.translate('shared.Shared.EventsView.InputPlaceholder.SearchByFieldsInPolicy');
            case ViewMode.Groups:
                return this.transloco.translate('shared.Shared.EventsView.InputPlaceholder.SearchByFieldsInGroup');
            default:
                return this.transloco.translate('shared.Shared.EventsView.InputPlaceholder.SearchByFields');
        }
    }

    saveState($event: WidthChangeEvent) {
        this.state.rightSidebar.width = $event.width;
    }

    onGridSort(sorting: Sorting) {
        this.sort.emit(sorting);
    }

    get isModulePage() {
        return this.hideModuleColumn;
    }
}
