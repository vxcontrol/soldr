import {
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
import { PopUpPlacements } from '@ptsecurity/mosaic/core';
import { BehaviorSubject, Subscription } from 'rxjs';

import { PERMISSIONS_TOKEN } from '@soldr/core';
import { Agent, AgentModule, AgentUpgradeTask } from '@soldr/models';
import { osList, ProxyPermission, Sorting, sortModules, sortTags } from '@soldr/shared';
import { SharedFacade } from '@soldr/store/shared';

import { LanguageService } from '../../services';
import { Filtration, ViewMode } from '../../types';
import { GridColumnFilterItem } from '../grid/grid.types';

const FIELD_FOR_TRANSLATE = ['status', 'os', 'auth_status'];

@Component({
    selector: 'soldr-agents-grid',
    templateUrl: './agents-grid.component.html',
    styleUrls: ['./agents-grid.component.scss']
})
export class AgentsGridComponent implements OnInit, OnChanges, OnDestroy {
    @Input() agents: Agent[];
    @Input() gridFilters: { [field: string]: GridColumnFilterItem[] };
    @Input() gridFiltrationByFields: Record<string, Filtration>;
    @Input() hidePoliciesColumn: boolean;
    @Input() isCancelUpgradingAgent: boolean;
    @Input() isLoading: boolean;
    @Input() isUpgradingAgents: boolean;
    @Input() searchValue: string;
    @Input() selectedAgent: Agent;
    @Input() selectedAgentModules: AgentModule[];
    @Input() stateStorageKey: string;
    @Input() stateStorageValue: any;
    @Input() storageGridKey: string;
    @Input() total: number;
    @Input() viewMode: ViewMode;

    @Output() changeSelection = new EventEmitter<Agent>();
    @Output() filter = new EventEmitter<Filtration>();
    @Output() loadNextPage = new EventEmitter();
    @Output() resetFiltration = new EventEmitter();
    @Output() search = new EventEmitter<string>();
    @Output() setTag = new EventEmitter<string>();
    @Output() sort = new EventEmitter<Sorting>();
    @Output() updateRow = new EventEmitter<Agent>();
    @Output() upgradeAgents = new EventEmitter<{ agents: Agent[]; version: string }>();
    @Output() cancelUpgradeAgent = new EventEmitter<{ hash: string; task: AgentUpgradeTask }>();

    agents$ = new BehaviorSubject<Agent[]>([]);
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
        groups: [],
        module: [],
        tags: []
    };
    gridFiltration: Filtration[] = [];
    language$ = this.languageService.current$;
    latestAgentBinaryVersion$ = this.sharedFacade.latestAgentBinaryVersion$;
    placement = PopUpPlacements;
    search$ = new BehaviorSubject<string>('');
    sortModules = sortModules;
    sortTags = sortTags;
    subscription = new Subscription();
    viewModeEnum = ViewMode;

    private inputIsUpgradingAgents$ = new BehaviorSubject<boolean>(false);
    private inputIsCancelUpgradingAgent$ = new BehaviorSubject<boolean>(false);

    constructor(
        private languageService: LanguageService,
        private sharedFacade: SharedFacade,
        private transloco: TranslocoService,
        @Inject(PERMISSIONS_TOKEN) public permitted: ProxyPermission
    ) {}

    ngOnInit(): void {
        FIELD_FOR_TRANSLATE.forEach(
            (field) =>
                (this.gridColumnsFilters[field] = this.gridColumnsFilters[field].map((filterItem) => ({
                    ...filterItem,
                    label: this.transloco.translate(filterItem.label)
                })))
        );
    }

    ngOnChanges({
        agents,
        searchValue,
        gridFilters,
        gridFiltrationByFields,
        isUpgradingAgents,
        isCancelUpgradingAgent
    }: SimpleChanges): void {
        if (agents?.currentValue) {
            this.agents$.next(this.agents);
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

        if (isUpgradingAgents) {
            this.inputIsUpgradingAgents$.next(isUpgradingAgents.currentValue as boolean);
        }

        if (isCancelUpgradingAgent) {
            this.inputIsCancelUpgradingAgent$.next(isCancelUpgradingAgent.currentValue as boolean);
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

    onGridSelectRows([agent]: Agent[]) {
        this.changeSelection.emit(agent);
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

    upgrade(event: { agents: Agent[]; version: string }) {
        this.upgradeAgents.emit({ agents: event.agents, version: event.version });
    }

    cancelUpgrade(event: { hash: string; task: AgentUpgradeTask }) {
        this.cancelUpgradeAgent.emit({ hash: event.hash, task: event.task });
    }

    afterUpgradeAgent() {
        this.updateRow.emit(this.selectedAgent);
    }
}
