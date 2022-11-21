import { Injectable } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslocoService } from '@ngneat/transloco';
import { Store } from '@ngrx/store';
import { combineLatest, map } from 'rxjs';

import { AgentsService, allListQuery } from '@soldr/api';
import { Agent, AgentUpgradeTask } from '@soldr/models';
import {
    AgentVersionPipe,
    Filter,
    Filtration,
    GridColumnFilterItem,
    LanguageService,
    osList,
    Sorting
} from '@soldr/shared';
import { SharedFacade } from '@soldr/store/shared';

import * as AgentListActions from './agent-list.actions';
import { State } from './agent-list.reducer';
import {
    selectAgentModules,
    selectAgents,
    selectDeleteError,
    selectFilterItemsGroupIds,
    selectFilterItemsModuleNames,
    selectFilterItemsOs,
    selectFilterItemsTags,
    selectFilterItemsVersions,
    selectFiltersWithCounter,
    selectGridFiltration,
    selectGridFiltrationByField,
    selectGridSearch,
    selectGridSorting,
    selectInitialized,
    selectInitializedAgent,
    selectInitialListQuery,
    selectIsBlockingAgents,
    selectIsDeletingAgents,
    selectIsDeletingFromGroup,
    selectIsLoadingAgent,
    selectIsLoadingAgents,
    selectIsMovingAgents,
    selectIsUpdatingAgent,
    selectIsUpdatingAgentData,
    selectIsUpgradingAgents,
    selectIsUpgradingCancelAgent,
    selectMoveToGroupError,
    selectPage,
    selectRestored,
    selectSelectedAgent,
    selectSelectedAgents,
    selectSelectedAgentsIds,
    selectSelectedFilterId,
    selectSelectedGroupId,
    selectTotal,
    selectUpdateError,
    selectVersions
} from './agent-list.selectors';

@Injectable({
    providedIn: 'root'
})
export class AgentListFacade {
    agent$ = this.store.select(selectSelectedAgent);
    agentModules$ = this.store.select(selectAgentModules);
    agents$ = this.store.select(selectAgents);
    allListQuery$ = this.store.select(selectInitialListQuery).pipe(map((query) => allListQuery(query)));
    deleteError$ = this.store.select(selectDeleteError);
    filters$ = this.store.select(selectFiltersWithCounter);
    gridFiltration$ = this.store.select(selectGridFiltration);
    gridFiltrationByField$ = this.store.select(selectGridFiltrationByField);
    isBlockingAgents$ = this.store.select(selectIsBlockingAgents);
    isCancelUpgradingAgent$ = this.store.select(selectIsUpgradingCancelAgent);
    isDeletingAgents$ = this.store.select(selectIsDeletingAgents);
    isDeletingFromGroup$ = this.store.select(selectIsDeletingFromGroup);
    isInitialized$ = this.store.select(selectInitialized);
    isInitializedAgent$ = this.store.select(selectInitializedAgent);
    isLoading$ = this.store.select(selectIsLoadingAgents);
    isLoadingAgent$ = this.store.select(selectIsLoadingAgent);
    isMovingAgents$ = this.store.select(selectIsMovingAgents);
    isRestored$ = this.store.select(selectRestored);
    isUpdatingAgent$ = this.store.select(selectIsUpdatingAgent);
    isUpdatingAgentData$ = this.store.select(selectIsUpdatingAgentData);
    isUpgradingAgents$ = this.store.select(selectIsUpgradingAgents);
    moveToGroupError$ = this.store.select(selectMoveToGroupError);
    page$ = this.store.select(selectPage);
    search$ = this.store.select(selectGridSearch);
    selectedAgents$ = this.store.select(selectSelectedAgents);
    selectedAgentsIds$ = this.store.select(selectSelectedAgentsIds);
    selectedFilterId$ = this.store.select(selectSelectedFilterId);
    selectedGroupId$ = this.store.select(selectSelectedGroupId);
    sorting$ = this.store.select(selectGridSorting);
    total$ = this.store.select(selectTotal);
    updateError$ = this.store.select(selectUpdateError);
    versions$ = this.store.select(selectVersions);

    gridFilterItems$ = combineLatest([
        this.store.select(selectFilterItemsGroupIds),
        this.sharedFacade.allGroups$,
        this.store.select(selectFilterItemsModuleNames),
        this.sharedFacade.allModules$,
        this.store.select(selectFilterItemsVersions),
        this.store.select(selectFilterItemsOs),
        this.store.select(selectFilterItemsTags)
    ]).pipe(
        map(([groupIds, groups, moduleNames, modules, versions, os, tags]) => ({
            groups: groupIds?.map((id) => groups.find((group) => group.id === parseInt(id))) || [],
            modules: moduleNames?.map((name) => modules.find((item) => item.info.name === name)) || [],
            versions,
            os,
            tags
        }))
    );
    gridColumnFilterItems$ = this.gridFilterItems$.pipe(
        map(({ groups, modules, versions, os, tags }): { [field: string]: GridColumnFilterItem[] } => ({
            groups: groups.map((group) => ({
                label: group?.info.name[this.languageService.lang],
                value: group?.id
            })),
            modules: modules.map((module) => ({
                label: module?.locale.module[this.languageService.lang].title || '',
                value: module?.info.name
            })),
            versions: versions?.map((version: string) => {
                const formatted = new AgentVersionPipe().transform(version);

                return { label: formatted, value: version } as GridColumnFilterItem;
            }),
            os: osList
                .filter(({ value }) => os?.find((filterItem) => filterItem === value))
                .map((osItem) => ({ ...osItem, label: this.transloco.translate(osItem.label) })),
            tags: tags?.map((tag) => ({ label: tag, value: tag }))
        }))
    );

    constructor(
        private activatedRoute: ActivatedRoute,
        private agentsService: AgentsService,
        private languageService: LanguageService,
        private router: Router,
        private sharedFacade: SharedFacade,
        private store: Store<State>,
        private transloco: TranslocoService
    ) {}

    updateFilters(filters: Filter[]): void {
        this.store.dispatch(AgentListActions.updateFilters({ filters }));
    }

    fetchFiltersCounters(): void {
        this.store.dispatch(AgentListActions.fetchCountersByFilters());
    }

    fetchFilterItems() {
        this.store.dispatch(AgentListActions.fetchFilterItems());
    }

    fetchAgentsPage(page?: number): void {
        this.store.dispatch(AgentListActions.fetchAgentsPage({ page }));
    }

    upgradeAgents(agents: Agent[], version: string) {
        const upgradingAgents = agents.filter(
            (agent) => agent.auth_status === 'authorized' && agent.version !== version
        );
        this.store.dispatch(AgentListActions.upgradeAgents({ agents: upgradingAgents, version }));
    }

    cancelUpgradeAgent(hash: string, task: AgentUpgradeTask) {
        this.store.dispatch(AgentListActions.cancelUpgradeAgent({ hash, task }));
    }

    fetchVersions(): void {
        this.store.dispatch(AgentListActions.fetchAgentsVersions());
    }

    selectFilter(id: string): void {
        this.store.dispatch(AgentListActions.selectFilter({ id }));
    }

    selectGroup(id: string): void {
        this.store.dispatch(AgentListActions.selectGroup({ id }));
    }

    setGridFiltration(filtration: Filtration): void {
        this.store.dispatch(AgentListActions.setGridFiltration({ filtration }));
    }

    setGridFiltrationByTag(tag: string): void {
        this.store.dispatch(AgentListActions.setGridFiltrationByTag({ tag }));
    }

    setGridSearch(value: string): void {
        this.store.dispatch(AgentListActions.setGridSearch({ value }));
    }

    resetFiltration(): void {
        this.store.dispatch(AgentListActions.resetFiltration());
    }

    setGridSorting(sorting: Sorting): void {
        this.store.dispatch(AgentListActions.setGridSorting({ sorting }));
    }

    restoreState(): void {
        const params = this.activatedRoute.snapshot.queryParams as {
            groupId: string;
            filterId: string;
            filtration: string;
            search: string;
            sort: string;
        };

        if (params.groupId) {
            this.selectGroup(params.groupId);
        }
        if (params.filterId) {
            this.selectFilter(params.filterId);
        }
        if (!params.groupId && !params.filterId) {
            this.selectFilter('authorized');
        }

        const gridFiltration = params.filtration ? JSON.parse(params.filtration) : [];
        const gridSearch = params.search || '';
        const sorting = params.sort ? JSON.parse(params.sort) : {};

        this.store.dispatch(AgentListActions.restoreState({ restoredState: { gridFiltration, gridSearch, sorting } }));
    }

    selectAgents(agents: Agent[]) {
        this.store.dispatch(AgentListActions.selectAgents({ agents }));
    }

    moveToGroups(ids: number[], groupId: number): void {
        this.store.dispatch(AgentListActions.moveAgentsToGroup({ ids, groupId }));
    }

    moveToNewGroups(ids: number[], groupName: string): void {
        this.store.dispatch(AgentListActions.moveAgentsToNewGroup({ ids, groupName }));
    }

    updateAgent(agent: Agent) {
        this.store.dispatch(AgentListActions.updateAgent({ agent }));
    }

    blockAgents(ids: number[]) {
        this.store.dispatch(AgentListActions.blockAgents({ ids }));
    }

    deleteAgents(ids: number[]) {
        this.store.dispatch(AgentListActions.deleteAgents({ ids }));
    }

    updateAgentsData(agents: Agent[]): void {
        this.store.dispatch(AgentListActions.updateAgentData({ agents }));
    }

    resetAgentErrors() {
        this.store.dispatch(AgentListActions.resetAgentErrors());
    }

    reset() {
        this.store.dispatch(AgentListActions.reset());
    }
}
