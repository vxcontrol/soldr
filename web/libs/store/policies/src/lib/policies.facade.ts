import { Injectable } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { TranslocoService } from '@ngneat/transloco';
import { Store } from '@ngrx/store';
import { catchError, combineLatest, filter, from, map, Observable, of, switchMap, toArray } from 'rxjs';

import {
    allListQuery,
    ModelsGroup,
    ModelsPolicy,
    PoliciesService,
    PrivatePolicies,
    PrivatePolicyInfo,
    SuccessResponse
} from '@soldr/api';
import { Agent, AgentUpgradeTask, Policy } from '@soldr/models';
import { Filter, Filtration, GridColumnFilterItem, LanguageService, osList, Sorting } from '@soldr/shared';
import { ModuleListFacade } from '@soldr/store/modules';
import { ModulesInstancesFacade } from '@soldr/store/modules-instances';
import { SharedFacade } from '@soldr/store/shared';

import * as PoliciesActions from './policies.actions';
import { State } from './policies.reducer';
import {
    selectPolicies,
    selectGridFiltration,
    selectGridFiltrationByField,
    selectGridSearch,
    selectGridSorting,
    selectIsCopyingPolicy,
    selectIsCreatingPolicy,
    selectIsDeletingPolicy,
    selectIsLinkingPolicy,
    selectIsLoadingPolicy,
    selectIsLoadingPolicies,
    selectIsUnlinkingPolicy,
    selectIsUpdatingPolicy,
    selectPage,
    selectRestored,
    selectSelectedPolicy,
    selectSelectedPolicyId,
    selectSelectedPolicies,
    selectSelectedPoliciesIds,
    selectTotal,
    selectSelectedFilterId,
    selectFiltersWithCounter,
    selectSelectedGroupId,
    selectInitialized,
    selectPolicyModules,
    selectPolicy,
    selectTotalEvents,
    selectPolicyEvents,
    selectEventsGridFiltration,
    selectEventsGridFiltrationByField,
    selectEventsPage,
    selectEventsGridSearch,
    selectIsLoadingEvents,
    selectAgentsGridFiltration,
    selectAgentsGridFiltrationByField,
    selectAgentsPage,
    selectAgentsGridSearch,
    selectTotalAgents,
    selectPolicyAgents,
    selectIsLoadingAgents,
    selectSelectedAgent,
    selectSelectedAgentId,
    selectGroupsGridFiltration,
    selectGroupsGridFiltrationByField,
    selectGroupsPage,
    selectGroupsGridSearch,
    selectTotalGroups,
    selectPolicyGroups,
    selectSelectedPolicyGroup,
    selectSelectedPolicyGroupId,
    selectIsLoadingGroups,
    selectIsLoadingModules,
    selectIsUpdatingAgentData,
    selectCopyError,
    selectCreateError,
    selectDeleteError,
    selectLinkPolicyFromGroupError,
    selectUpdateError,
    selectUnlinkPolicyFromGroupError,
    selectInitialListQuery,
    selectIsUpgradingAgents,
    selectIsCancelUpgradingAgent,
    selectFilterItemsGroupIds,
    selectFilterItemsModuleNames,
    selectFilterItemsTags,
    selectAgentFilterItemGroupIds,
    selectAgentFilterItemTags,
    selectAgentFilterItemOs,
    selectAgentFilterItemModuleNames,
    selectEventFilterItemGroupIds,
    selectEventFilterItemModuleIds,
    selectEventFilterItemAgentIds,
    selectGroupFilterItemTags,
    selectGroupFilterItemModuleNames,
    selectGroupFilterItemPolicyIds,
    selectCreatedPolicy
} from './policies.selectors';

@Injectable({
    providedIn: 'root'
})
export class PoliciesFacade {
    agentsGridFiltration$ = this.store.select(selectAgentsGridFiltration);
    agentsGridFiltrationByFields$ = this.store.select(selectAgentsGridFiltrationByField);
    agentsPage$ = this.store.select(selectAgentsPage);
    agentsSearchValue$ = this.store.select(selectAgentsGridSearch);
    agentsTotal$ = this.store.select(selectTotalAgents);
    allListQuery$ = this.store.select(selectInitialListQuery).pipe(map((initialQuery) => allListQuery(initialQuery)));
    copyError$ = this.store.select(selectCopyError);
    createError$ = this.store.select(selectCreateError);
    createdPolicy$ = this.store.select(selectCreatedPolicy);
    deleteError$ = this.store.select(selectDeleteError);
    eventsGridFiltration$ = this.store.select(selectEventsGridFiltration);
    eventsGridFiltrationByFields$ = this.store.select(selectEventsGridFiltrationByField);
    eventsPage$ = this.store.select(selectEventsPage);
    eventsSearchValue$ = this.store.select(selectEventsGridSearch);
    eventsTotal$ = this.store.select(selectTotalEvents);
    filters$ = this.store.select(selectFiltersWithCounter);
    gridFiltration$ = this.store.select(selectGridFiltration);
    gridFiltrationByField$ = this.store.select(selectGridFiltrationByField);
    groupsGridFiltration$ = this.store.select(selectGroupsGridFiltration);
    groupsGridFiltrationByFields$ = this.store.select(selectGroupsGridFiltrationByField);
    groupsPage$ = this.store.select(selectGroupsPage);
    groupsSearchValue$ = this.store.select(selectGroupsGridSearch);
    groupsTotal$ = this.store.select(selectTotalGroups);
    isCopyingPolicy$ = this.store.select(selectIsCopyingPolicy);
    isCreatingPolicy$ = this.store.select(selectIsCreatingPolicy);
    isDeletingPolicy$ = this.store.select(selectIsDeletingPolicy);
    isInitialized$ = this.store.select(selectInitialized);
    isLinkingPolicy$ = this.store.select(selectIsLinkingPolicy);
    isLoadingAgents$ = this.store.select(selectIsLoadingAgents);
    isLoadingEvents$ = this.store.select(selectIsLoadingEvents);
    isLoadingGroups$ = this.store.select(selectIsLoadingGroups);
    isLoadingModules$ = this.store.select(selectIsLoadingModules);
    isLoadingPolicies$ = this.store.select(selectIsLoadingPolicies);
    isLoadingPolicy$ = this.store.select(selectIsLoadingPolicy);
    isRestored$ = this.store.select(selectRestored);
    isUnlinkingPolicy$ = this.store.select(selectIsUnlinkingPolicy);
    isUpdatingAgentData$ = this.store.select(selectIsUpdatingAgentData);
    isUpdatingPolicy$ = this.store.select(selectIsUpdatingPolicy);
    linkPolicyFromGroupError$ = this.store.select(selectLinkPolicyFromGroupError);
    page$ = this.store.select(selectPage);
    policies$ = this.store.select(selectPolicies);
    policy$ = this.store.select(selectPolicy);
    policyAgents$ = this.store.select(selectPolicyAgents);
    policyEvents$ = this.store.select(selectPolicyEvents);
    policyModules$ = this.store.select(selectPolicyModules);
    policyGroups$ = this.store.select(selectPolicyGroups);
    search$ = this.store.select(selectGridSearch);
    selectedFilterId$ = this.store.select(selectSelectedFilterId);
    selectedGroupId$ = this.store.select(selectSelectedGroupId);
    selectedAgent$ = this.store.select(selectSelectedAgent);
    selectedAgentId$ = this.store.select(selectSelectedAgentId);
    selectedPolicies$ = this.store.select(selectSelectedPolicies);
    selectedPoliciesIds$ = this.store.select(selectSelectedPoliciesIds);
    selectedPolicy$ = this.store.select(selectSelectedPolicy);
    selectedPolicyId$ = this.store.select(selectSelectedPolicyId);
    selectedPolicyGroupId$ = this.store.select(selectSelectedPolicyGroupId);
    selectedPolicyGroup$ = this.store.select(selectSelectedPolicyGroup);
    sorting$ = this.store.select(selectGridSorting);
    total$ = this.store.select(selectTotal);
    unlinkPolicyFromGroupError$ = this.store.select(selectUnlinkPolicyFromGroupError);
    updateError$ = this.store.select(selectUpdateError);

    isUpgradingAgents$ = this.store.select(selectIsUpgradingAgents);
    isCancelUpgradingAgent$ = this.store.select(selectIsCancelUpgradingAgent);
    gridFilterItems$ = combineLatest([
        this.store.select(selectFilterItemsGroupIds),
        this.sharedFacade.allGroups$,
        this.store.select(selectFilterItemsModuleNames),
        this.policyModules$,
        this.store.select(selectFilterItemsTags)
    ]).pipe(
        map(([groupIds, groups, moduleNames, modules, tags]) => ({
            groups: groupIds?.map((id) => groups.find((group) => group.id === parseInt(id))) || [],
            modules: moduleNames?.map((name) => modules.find((item) => item.info.name === name)) || [],
            tags
        }))
    );
    gridColumnFilterItems$ = this.gridFilterItems$.pipe(
        map(({ groups, modules, tags }): { [field: string]: GridColumnFilterItem[] } => ({
            groups: groups.map((group) => ({
                label: group?.info.name[this.languageService.lang],
                value: group?.id
            })),
            modules: modules.map((module) => ({
                label: module?.locale.module[this.languageService.lang].title || '',
                value: module?.info.name
            })),
            tags: tags?.map((tag) => ({ label: tag, value: tag }))
        }))
    );
    agentGridFilterItems$ = combineLatest([
        this.sharedFacade.allGroups$,
        this.sharedFacade.allModules$,
        this.store.select(selectAgentFilterItemGroupIds),
        this.store.select(selectAgentFilterItemModuleNames),
        this.store.select(selectAgentFilterItemOs),
        this.store.select(selectAgentFilterItemTags)
    ]).pipe(
        map(([allGroups, modules, groupIds, moduleNames, os, tags]) => ({
            groups: groupIds?.map((id) => allGroups.find((group) => group.id === parseInt(id))) || [],
            modules: moduleNames?.map((name) => modules.find((item) => item.info.name === name)) || [],
            os,
            tags
        }))
    );
    agentGridColumnFilterItems$ = this.agentGridFilterItems$.pipe(
        map(({ groups, modules, os, tags }): { [field: string]: GridColumnFilterItem[] } => ({
            groups: groups.map((group) => ({
                label: group?.info.name[this.languageService.lang],
                value: group?.id
            })),
            modules: modules.map((module) => ({
                label: module?.locale.module[this.languageService.lang].title,
                value: module?.info.name
            })),
            os: osList
                .filter(({ value }) => os?.find((filterItem) => filterItem === value))
                .map((osItem) => ({ ...osItem, label: this.transloco.translate(osItem.label) })),
            tags: tags?.map((tag) => ({ label: tag, value: tag }))
        }))
    );
    eventGridFilterItems$ = combineLatest([
        this.policyModules$,
        this.sharedFacade.allAgents$,
        this.sharedFacade.allGroups$,
        this.store.select(selectEventFilterItemModuleIds),
        this.store.select(selectEventFilterItemAgentIds),
        this.store.select(selectEventFilterItemGroupIds)
    ]).pipe(
        filter(([modules, allAgents, allGroups]) => !!modules.length && !!allAgents.length && !!allGroups.length),
        map(([modules, allAgents, allGroups, moduleIds, agentIds, groupIds]) => ({
            agents: agentIds?.map((id) => allAgents.find((agent) => agent.id === parseInt(id))) || [],
            modules: moduleIds?.map((id) => modules.find((module) => module.id === parseInt(id))) || [],
            groups: groupIds?.map((id) => allGroups.find((policy) => policy.id === parseInt(id))) || []
        }))
    );
    eventGridColumnFilterItems$ = this.eventGridFilterItems$.pipe(
        map(({ agents, modules, groups }): { [field: string]: GridColumnFilterItem[] } => ({
            agents: agents.map((agent) => ({
                label: agent?.description,
                value: agent?.id
            })),
            modules: modules.map((module) => ({
                label: module?.locale.module[this.languageService.lang].title,
                value: module?.info.name
            })),
            groups: groups.map((group) => ({
                label: group?.info.name[this.languageService.lang],
                value: group?.id
            }))
        }))
    );
    gridGroupFilterItems$ = combineLatest([
        this.sharedFacade.allPolicies$,
        this.sharedFacade.allModules$,
        this.store.select(selectGroupFilterItemPolicyIds),
        this.store.select(selectGroupFilterItemModuleNames),
        this.store.select(selectGroupFilterItemTags)
    ]).pipe(
        filter(([allPolicies, modules]) => !!allPolicies.length && !!modules.length),
        map(([allPolicies, modules, policyIds, moduleNames, tags]) => ({
            modules: moduleNames?.map((name) => modules.find((module) => module.info.name === name)) || [],
            policies: policyIds?.map((id) => allPolicies.find((policy) => policy.id === parseInt(id))) || [],
            tags: tags?.map((tag) => ({ label: tag, value: tag })) || []
        }))
    );
    gridGroupColumnFilterItems$ = this.gridGroupFilterItems$.pipe(
        map(({ modules, policies, tags }): { [field: string]: GridColumnFilterItem[] } => ({
            modules: modules.map((module) => ({
                label: module.locale.module[this.languageService.lang].title,
                value: module.info.name
            })),
            policies: policies.map((policy) => ({
                label: policy.info.name[this.languageService.lang],
                value: policy.id
            })),
            tags
        }))
    );
    moduleEventsGridFilterItems$ = combineLatest([
        this.sharedFacade.allAgents$,
        this.sharedFacade.allGroups$,
        this.modulesInstancesFacade.moduleEventsFilterItemAgentIds$,
        this.modulesInstancesFacade.moduleEventsFilterItemGroupIds$
    ]).pipe(
        map(([allAgents, allGroups, agentIds, groupIds]) => ({
            agents: agentIds?.map((id: string) => allAgents?.find((agent) => agent.id === parseInt(id))),
            groups: groupIds?.map((id: string) => allGroups?.find((group) => group.id === parseInt(id)))
        }))
    );
    moduleEventsGridColumnFilterItems$ = this.moduleEventsGridFilterItems$.pipe(
        map(({ agents, groups }) => ({
            agents: agents?.map((agent) => ({
                label: agent?.description,
                value: agent?.id
            })),
            groups: groups?.map((group) => ({
                label: group?.info.name[this.languageService.lang],
                value: group?.id
            }))
        }))
    );

    constructor(
        private activatedRoute: ActivatedRoute,
        private languageService: LanguageService,
        private modulesInstancesFacade: ModulesInstancesFacade,
        private policiesService: PoliciesService,
        private sharedFacade: SharedFacade,
        private store: Store<State>,
        private transloco: TranslocoService
    ) {}

    updateFilters(filters: Filter[]): void {
        this.store.dispatch(PoliciesActions.updateFilters({ filters }));
    }

    selectFilter(id: string): void {
        this.store.dispatch(PoliciesActions.selectFilter({ id }));
    }

    selectGroup(id: string): void {
        this.store.dispatch(PoliciesActions.selectGroup({ id }));
    }

    fetchFiltersCounters(): void {
        this.store.dispatch(PoliciesActions.fetchCountersByFilters());
    }

    fetchPoliciesPage(page?: number) {
        this.store.dispatch(PoliciesActions.fetchPoliciesPage({ page }));
    }

    fetchFilterItems() {
        this.store.dispatch(PoliciesActions.fetchFilterItems());
    }

    fetchAgentFilterItems() {
        this.store.dispatch(PoliciesActions.fetchAgentFilterItems());
    }

    fetchEventFilterItems() {
        this.store.dispatch(PoliciesActions.fetchEventFilterItems());
    }

    fetchGroupFilterItems() {
        this.store.dispatch(PoliciesActions.fetchGroupFilterItems());
    }

    selectPolicy(id: string): void {
        this.store.dispatch(PoliciesActions.selectPolicy({ id }));
    }

    selectPolicies(policies: Policy[]) {
        this.store.dispatch(PoliciesActions.selectPolicies({ policies }));
    }

    createPolicy(policy: PrivatePolicyInfo): void {
        this.store.dispatch(PoliciesActions.createPolicy({ policy }));
    }

    copyPolicy(policy: Policy, redirect: boolean): void {
        this.store.dispatch(PoliciesActions.copyPolicy({ policy, redirect }));
    }

    updatePolicy(policy: Policy): void {
        this.store.dispatch(PoliciesActions.updatePolicy({ policy }));
    }

    deletePolicy(hash: string): void {
        this.store.dispatch(PoliciesActions.deletePolicy({ hash }));
    }

    linkPolicyToGroup(hash: string, group: ModelsGroup): void {
        this.store.dispatch(PoliciesActions.linkPolicyToGroup({ hash, group }));
    }

    unlinkPolicyFromGroup(hash: string, group: ModelsGroup): void {
        this.store.dispatch(PoliciesActions.unlinkPolicyFromGroup({ hash, group }));
    }

    setGridFiltration(filtration: Filtration): void {
        this.store.dispatch(PoliciesActions.setGridFiltration({ filtration }));
    }

    setGridFiltrationByTag(tag: string): void {
        this.store.dispatch(PoliciesActions.setGridFiltrationByTag({ tag }));
    }

    setGridSearch(value: string): void {
        this.store.dispatch(PoliciesActions.setGridSearch({ value }));
    }

    resetFiltration(): void {
        this.store.dispatch(PoliciesActions.resetFiltration());
    }

    setGridSorting(sorting: Sorting | Record<never, any>): void {
        this.store.dispatch(PoliciesActions.setGridSorting({ sorting }));
    }

    restoreState(): void {
        const params = this.activatedRoute.snapshot.queryParams as Record<string, string>;

        if (params.groupId) {
            this.selectGroup(params.groupId);
        }
        if (params.filterId) {
            this.selectFilter(params.filterId);
        }
        if (!params.groupId && !params.filterId) {
            this.selectFilter('all_policies');
        }

        const gridFiltration: Filtration[] = params.filtration ? JSON.parse(params.filtration) : [];
        const gridSearch: string = params.search || '';
        const sorting: Sorting = params.sort ? JSON.parse(params.sort) : {};

        this.store.dispatch(PoliciesActions.restoreState({ restoredState: { gridFiltration, gridSearch, sorting } }));
    }

    getIsExistedPoliciesByName(name: string, exclude: string[]): Observable<boolean> {
        const query = allListQuery({
            filters: [
                {
                    field: 'name',
                    value: [name]
                }
            ]
        });

        return this.policiesService.fetchList(query).pipe(
            switchMap((response: SuccessResponse<PrivatePolicies>) => from(response.data?.policies)),
            filter(
                (policy: ModelsPolicy) =>
                    !exclude.some((value) => [policy.info.name.ru, policy.info.name.en].includes(value))
            ),
            toArray(),
            map((policies) => policies.length > 0),
            catchError(() => of(false))
        );
    }

    fetchPolicy(hash: string): void {
        this.store.dispatch(PoliciesActions.fetchPolicy({ hash }));
    }

    fetchModules(hash: string): void {
        this.store.dispatch(PoliciesActions.fetchPolicyModules({ hash }));
    }

    fetchEvents(id: number, page?: number): void {
        this.store.dispatch(PoliciesActions.fetchPolicyEvents({ id, page }));
    }

    setEventsGridFiltration(filtration: Filtration): void {
        this.store.dispatch(PoliciesActions.setEventsGridFiltration({ filtration }));
    }

    setEventsGridSearch(value: string): void {
        this.store.dispatch(PoliciesActions.setEventsGridSearch({ value }));
    }

    resetEventsFiltration(): void {
        this.store.dispatch(PoliciesActions.resetEventsFiltration());
    }

    setEventsGridSorting(sorting: Sorting): void {
        this.store.dispatch(PoliciesActions.setEventsGridSorting({ sorting }));
    }

    fetchAgents(id: number, page?: number): void {
        this.store.dispatch(PoliciesActions.fetchPolicyAgents({ id, page }));
    }

    setAgentsGridFiltration(filtration: Filtration): void {
        this.store.dispatch(PoliciesActions.setAgentsGridFiltration({ filtration }));
    }

    setAgentsGridSearch(value: string): void {
        this.store.dispatch(PoliciesActions.setAgentsGridSearch({ value }));
    }

    resetAgentsFiltration(): void {
        this.store.dispatch(PoliciesActions.resetAgentsFiltration());
    }

    setAgentsGridSorting(sorting: Sorting): void {
        this.store.dispatch(PoliciesActions.setAgentsGridSorting({ sorting }));
    }

    selectAgent(id: number): void {
        this.store.dispatch(PoliciesActions.selectAgent({ id }));
    }

    fetchGroups(id: number, page?: number): void {
        this.store.dispatch(PoliciesActions.fetchPolicyGroups({ id, page }));
    }

    setGroupsGridFiltration(filtration: Filtration): void {
        this.store.dispatch(PoliciesActions.setGroupsGridFiltration({ filtration }));
    }

    setGroupsGridSearch(value: string): void {
        this.store.dispatch(PoliciesActions.setGroupsGridSearch({ value }));
    }

    resetGroupsFiltration(): void {
        this.store.dispatch(PoliciesActions.resetGroupsFiltration());
    }

    setGroupsGridSorting(sorting: Sorting): void {
        this.store.dispatch(PoliciesActions.setGroupsGridSorting({ sorting }));
    }

    selectPolicyGroup(id: number): void {
        this.store.dispatch(PoliciesActions.selectPolicyGroup({ id }));
    }

    updateAgentData(agent: Agent): void {
        this.store.dispatch(PoliciesActions.updateAgentData({ agent }));
    }

    upgradeAgents(agents: Agent[], version: string) {
        const upgradingAgents = agents.filter(
            (agent) => agent.auth_status === 'authorized' && agent.version !== version
        );
        this.store.dispatch(PoliciesActions.upgradeAgents({ agents: upgradingAgents, version }));
    }

    cancelUpgradeAgent(hash: string, task: AgentUpgradeTask) {
        this.store.dispatch(PoliciesActions.cancelUpgradeAgent({ hash, task }));
    }

    resetCreatedPolicy() {
        this.store.dispatch(PoliciesActions.resetCreatedPolicy());
    }

    resetPolicyErrors() {
        this.store.dispatch(PoliciesActions.resetPolicyErrors());
    }

    reset() {
        this.store.dispatch(PoliciesActions.reset());
    }
}
