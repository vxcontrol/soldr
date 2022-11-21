import { Component, EventEmitter, Input, OnChanges, OnDestroy, Output, SimpleChanges } from '@angular/core';
import { PopUpPlacements } from '@ptsecurity/mosaic/core';
import { BehaviorSubject, Subscription } from 'rxjs';

import { Group, GroupModule, PolicyModule } from '@soldr/models';
import { sortTags, sortModules, sortPolicies, Sorting } from '@soldr/shared';
import { SharedFacade } from '@soldr/store/shared';

import { LanguageService } from '../../services';
import { Filtration, osList, ViewMode } from '../../types';
import { GridColumnFilterItem } from '../grid/grid.types';

@Component({
    selector: 'soldr-groups-grid',
    templateUrl: './groups-grid.component.html',
    styleUrls: ['./groups-grid.component.scss']
})
export class GroupsGridComponent implements OnChanges, OnDestroy {
    @Input() groups: Group[];
    @Input() gridFilters: { [field: string]: GridColumnFilterItem[] };
    @Input() gridFiltrationByFields: Record<string, Filtration>;
    @Input() hidePoliciesColumn: boolean;
    @Input() isLoading: boolean;
    @Input() modules: PolicyModule[] = [];
    @Input() searchValue: string;
    @Input() selectedGroup: Group;
    @Input() selectedGroupModules: GroupModule[];
    @Input() storageGridKey: string;
    @Input() stateStorageKey: string;
    @Input() stateStorageValue: any;
    @Input() total: number;
    @Input() viewMode: ViewMode;

    @Output() changeSelection = new EventEmitter<Group>();
    @Output() filter = new EventEmitter<Filtration>();
    @Output() loadNextPage = new EventEmitter();
    @Output() search = new EventEmitter<string>();
    @Output() resetFiltration = new EventEmitter();
    @Output() setTag = new EventEmitter<string>();
    @Output() sort = new EventEmitter<Sorting>();

    groups$ = new BehaviorSubject<Group[]>([]);
    gridColumnsFilters: { [field: string]: GridColumnFilterItem[] } = {
        status: [
            {
                label: 'agents.Agents.AgentsList.DropdownItemText.Connected',
                value: 'connected'
            },
            {
                label: 'agents.Agents.AgentsList.DropdownItemText.Disconnected',
                value: 'disconnected'
            },
            {
                label: 'shared.Shared.Pseudo.DropdownItemText.Any',
                value: undefined
            }
        ],
        os: [...osList],
        // eslint-disable-next-line @typescript-eslint/naming-convention
        auth_status: [
            {
                label: 'agents.Agents.AgentsList.DropdownItemText.Authorized',
                value: 'authorized'
            },
            {
                label: 'agents.Agents.AgentsList.DropdownItemText.Unauthorized',
                value: 'unauthorized'
            },
            {
                label: 'agents.Agents.AgentsList.DropdownItemText.Blocked',
                value: 'blocked'
            },
            {
                label: 'shared.Shared.Pseudo.DropdownItemText.Any',
                value: undefined
            }
        ],
        group_id: []
    };
    gridFiltration: Filtration[] = [];
    language$ = this.languageService.current$;
    placement = PopUpPlacements;
    search$ = new BehaviorSubject<string>('');
    sortModules = sortModules;
    sortPolicies = sortPolicies;
    sortTags = sortTags;
    subscription = new Subscription();
    viewModeEnum = ViewMode;

    constructor(private languageService: LanguageService, private sharedFacade: SharedFacade) {
        this.sharedFacade.fetchAllPolicies();
        this.sharedFacade.fetchAllModules();
    }

    ngOnChanges({ groups, searchValue, gridFilters, gridFiltrationByFields }: SimpleChanges): void {
        if (groups?.currentValue) {
            this.groups$.next(this.groups);
        }

        if (searchValue) {
            this.search$.next(this.searchValue || '');
        }

        if (gridFilters?.currentValue) {
            this.gridColumnsFilters = {
                status: [...this.gridColumnsFilters.status],
                auth_status: [...this.gridColumnsFilters.auth_status],
                ...this.gridFilters
            };
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

    onGridSelectRows([group]: Group[]) {
        this.changeSelection.emit(group);
    }

    onGridFilter(filtration: Filtration) {
        this.filter.next(filtration);
    }

    onResetFiltration() {
        this.resetFiltration.emit();
    }

    onSetTag(tag: string) {
        this.setTag.emit(tag);
    }

    onGridSort(sorting: Sorting) {
        this.sort.emit(sorting);
    }
}
