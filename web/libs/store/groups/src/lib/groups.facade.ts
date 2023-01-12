import { Injectable } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslocoService } from '@ngneat/transloco';
import { Store } from '@ngrx/store';
import { catchError, combineLatest, filter, from, map, Observable, of, switchMap, toArray } from 'rxjs';

import {
    allListQuery,
    GroupsService,
    ModelsGroup,
    ModelsPolicy,
    PrivateGroupInfo,
    PrivateGroups,
    SuccessResponse
} from '@soldr/api';
import { Agent, AgentUpgradeTask, Group } from '@soldr/models';
import { Filtration, GridColumnFilterItem, LanguageService, osList, Sorting } from '@soldr/shared';
import { ModulesInstancesFacade } from '@soldr/store/modules-instances';
import { SharedFacade } from '@soldr/store/shared';

import * as GroupsActions from './groups.actions';
import { State } from './groups.reducer';
import {
    selectGridFiltration,
    selectGridFiltrationByField,
    selectGridSearch,
    selectGridSorting,
    selectGroup,
    selectGroupAgents,
    selectGroupModules,
    selectGroups,
    selectIsCopyingGroup,
    selectIsCreatingGroup,
    selectIsDeletingGroup,
    selectIsLinkingGroup,
    selectIsLoadingGroup,
    selectIsLoadingGroups,
    selectIsUnlinkingGroup,
    selectIsUpdatingGroup,
    selectPage,
    selectRestored,
    selectSelectedGroup,
    selectSelectedGroupId,
    selectSelectedGroups,
    selectSelectedGroupsIds,
    selectTotal,
    selectAgentsGridSearch,
    selectAgentsGridFiltration,
    selectAgentsGridFiltrationByField,
    selectAgentsPage,
    selectSelectedAgent,
    selectTotalAgents,
    selectIsLoadingAgents,
    selectGroupEvents,
    selectEventsGridFiltrationByField,
    selectIsLoadingEvents,
    selectEventsGridSearch,
    selectTotalEvents,
    selectEventsPage,
    selectIsLoadingPolicies,
    selectGroupPolicies,
    selectTotalPolicies,
    selectPoliciesPage,
    selectPoliciesGridSearch,
    selectPoliciesGridFiltration,
    selectPoliciesGridFiltrationByField,
    selectSelectedPolicy,
    selectIsUpdatingAgentData,
    selectDeleteError,
    selectCopyError,
    selectUnlinkGroupFromPolicyError,
    selectLinkGroupToPolicyError,
    selectUpdateError,
    selectCreateError,
    selectInitialListQuery,
    selectIsUpgradingAgents,
    selectIsCancelUpgradingAgent,
    selectFilterItemsTags,
    selectFilterItemsModuleNames,
    selectFilterItemsPolicyIds,
    selectEventsGridFiltration,
    selectAgentFilterItemGroupIds,
    selectAgentFilterItemModuleNames,
    selectAgentFilterItemOs,
    selectAgentFilterItemTags,
    selectPolicyFilterItemModuleNames,
    selectPolicyFilterItemTags,
    selectEventFilterItemModuleIds,
    selectEventFilterItemAgentNames,
    selectEventFilterItemPolicyIds,
    selectCreatedGroup
} from './groups.selectors';

@Injectable({
    providedIn: 'root'
})
export class GroupsFacade {
    agentsGridFiltration$ = this.store.select(selectAgentsGridFiltration);
    agentsGridFiltrationByFields$ = this.store.select(selectAgentsGridFiltrationByField);
    agentsPage$ = this.store.select(selectAgentsPage);
    agentsSearchValue$ = this.store.select(selectAgentsGridSearch);
    agentsTotal$ = this.store.select(selectTotalAgents);
    allListQuery$ = this.store.select(selectInitialListQuery).pipe(map((initialQuery) => allListQuery(initialQuery)));
    copyError$ = this.store.select(selectCopyError);
    createError$ = this.store.select(selectCreateError);
    createdGroup$ = this.store.select(selectCreatedGroup);
    deleteError$ = this.store.select(selectDeleteError);
    eventsGridFiltration$ = this.store.select(selectEventsGridFiltration);
    eventsGridFiltrationByFields$ = this.store.select(selectEventsGridFiltrationByField);
    eventsPage$ = this.store.select(selectEventsPage);
    eventsSearchValue$ = this.store.select(selectEventsGridSearch);
    eventsTotal$ = this.store.select(selectTotalEvents);
    gridFiltration$ = this.store.select(selectGridFiltration);
    gridFiltrationByField$ = this.store.select(selectGridFiltrationByField);
    group$ = this.store.select(selectGroup);
    groupAgents$ = this.store.select(selectGroupAgents);
    groupEvents$ = this.store.select(selectGroupEvents);
    groupModules$ = this.store.select(selectGroupModules);
    groups$ = this.store.select(selectGroups);
    isCopyingGroup$ = this.store.select(selectIsCopyingGroup);
    isCreatingGroup$ = this.store.select(selectIsCreatingGroup);
    isDeletingGroup$ = this.store.select(selectIsDeletingGroup);
    isLinkingGroup$ = this.store.select(selectIsLinkingGroup);
    isLoadingAgents$ = this.store.select(selectIsLoadingAgents);
    isLoadingEvents$ = this.store.select(selectIsLoadingEvents);
    isLoadingGroup$ = this.store.select(selectIsLoadingGroup);
    isLoadingGroups$ = this.store.select(selectIsLoadingGroups);
    isLoadingPolicies$ = this.store.select(selectIsLoadingPolicies);
    isRestored$ = this.store.select(selectRestored);
    isUnlinkingGroup$ = this.store.select(selectIsUnlinkingGroup);
    isUpdatingAgentData$ = this.store.select(selectIsUpdatingAgentData);
    isUpdatingGroup$ = this.store.select(selectIsUpdatingGroup);
    linkGroupToPolicyError$ = this.store.select(selectLinkGroupToPolicyError);
    page$ = this.store.select(selectPage);
    policiesGridFiltrationByFields$ = this.store.select(selectPoliciesGridFiltrationByField);
    search$ = this.store.select(selectGridSearch);
    selectedAgent$ = this.store.select(selectSelectedAgent);
    selectedGroup$ = this.store.select(selectSelectedGroup);
    selectedGroupId$ = this.store.select(selectSelectedGroupId);
    selectedGroups$ = this.store.select(selectSelectedGroups);
    selectedGroupsIds$ = this.store.select(selectSelectedGroupsIds);
    selectedPolicy$ = this.store.select(selectSelectedPolicy);
    sorting$ = this.store.select(selectGridSorting);
    total$ = this.store.select(selectTotal);
    unlinkGroupFromPolicyError$ = this.store.select(selectUnlinkGroupFromPolicyError);
    updateError$ = this.store.select(selectUpdateError);

    groupPolicies$ = this.store.select(selectGroupPolicies);
    policiesTotal$ = this.store.select(selectTotalPolicies);
    policiesPage$ = this.store.select(selectPoliciesPage);
    policiesSearchValue$ = this.store.select(selectPoliciesGridSearch);
    policiesGridFiltration$ = this.store.select(selectPoliciesGridFiltration);

    isUpgradingAgents$ = this.store.select(selectIsUpgradingAgents);
    isCancelUpgradingAgent$ = this.store.select(selectIsCancelUpgradingAgent);

    gridFilterItems$ = combineLatest([
        this.sharedFacade.allPolicies$,
        this.sharedFacade.allModules$,
        this.store.select(selectFilterItemsPolicyIds),
        this.store.select(selectFilterItemsModuleNames),
        this.store.select(selectFilterItemsTags)
    ]).pipe(
        filter(([policies, modules]) => !!policies.length && !!modules.length),
        map(([policies, modules, policyIds, moduleNames, tags]) => ({
            policies: policyIds?.map((id) => policies.find((policy) => policy.id === parseInt(id))),
            modules: moduleNames?.map((name) => modules.find((module) => module.info.name === name)),
            tags
        }))
    );
    gridColumnFilterItems$ = this.gridFilterItems$.pipe(
        map(({ policies, modules, tags }): { [field: string]: GridColumnFilterItem[] } => ({
            modules: modules?.map((module) => ({
                label: module?.locale.module[this.languageService.lang].title,
                value: module?.info.name
            })),
            policies: policies?.map((policy) => ({
                label: policy?.info.name[this.languageService.lang],
                value: policy?.id
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
    policyGridFilterItems$ = combineLatest([
        this.sharedFacade.allModules$,
        this.store.select(selectPolicyFilterItemModuleNames),
        this.store.select(selectPolicyFilterItemTags)
    ]).pipe(
        map(([modules, moduleNames, tags]) => ({
            modules: moduleNames?.map((name) => modules.find((item) => item.info.name === name)) || [],
            tags
        }))
    );
    policyGridColumnFilterItems$ = this.policyGridFilterItems$.pipe(
        map(({ modules, tags }): { [field: string]: GridColumnFilterItem[] } => ({
            modules: modules.map((module) => ({
                label: module?.locale.module[this.languageService.lang].title,
                value: module?.info.name
            })),
            os: osList.map((osItem) => ({ ...osItem, label: this.transloco.translate(osItem.label) })),
            tags: tags?.map((tag) => ({ label: tag, value: tag }))
        }))
    );
    eventGridFilterItems$ = combineLatest([
        this.groupModules$,
        this.sharedFacade.allPolicies$,
        this.store.select(selectEventFilterItemModuleIds),
        this.store.select(selectEventFilterItemAgentNames),
        this.store.select(selectEventFilterItemPolicyIds)
    ]).pipe(
        filter(([modules, allPolicies]) => !!modules.length && !!allPolicies.length),
        map(([modules, allPolicies, moduleIds, agentNames, policyIds]) => ({
            agents: agentNames || [],
            modules: moduleIds?.map((id) => modules.find((module) => module.id === parseInt(id))) || [],
            policies: policyIds?.map((id) => allPolicies.find((policy) => policy.id === parseInt(id))) || []
        }))
    );
    eventGridColumnFilterItems$ = this.eventGridFilterItems$.pipe(
        map(({ agents, modules, policies }): { [field: string]: GridColumnFilterItem[] } => ({
            agents: agents.map((name) => ({
                label: name,
                value: name
            })),
            modules: modules.map((module) => ({
                label: module?.locale.module[this.languageService.lang].title,
                value: module?.info.name
            })),
            policies: policies.map((policy) => ({
                label: policy?.info.name[this.languageService.lang],
                value: policy?.id
            }))
        }))
    );
    moduleEventsGridFilterItems$ = this.modulesInstancesFacade.moduleEventsFilterItemAgentNames$.pipe(
        map((agentNames) => ({
            agents: agentNames || []
        }))
    );
    moduleEventsGridColumnFilterItems$ = this.moduleEventsGridFilterItems$.pipe(
        map(({ agents }) => ({
            agents: agents?.map((agentName) => ({
                label: agentName,
                value: agentName
            }))
        }))
    );

    constructor(
        private activatedRoute: ActivatedRoute,
        private groupsService: GroupsService,
        private languageService: LanguageService,
        private modulesInstancesFacade: ModulesInstancesFacade,
        private router: Router,
        private sharedFacade: SharedFacade,
        private store: Store<State>,
        private transloco: TranslocoService
    ) {}

    fetchGroupsPage(page?: number) {
        this.store.dispatch(GroupsActions.fetchGroupsPage({ page }));
    }

    fetchAgents(id: number, page?: number): void {
        this.store.dispatch(GroupsActions.fetchGroupAgents({ id, page }));
    }

    setAgentsGridSearch(value: string): void {
        this.store.dispatch(GroupsActions.setAgentsGridSearch({ value }));
    }

    setAgentsGridFiltration(filtration: Filtration): void {
        this.store.dispatch(GroupsActions.setAgentsGridFiltration({ filtration }));
    }

    resetAgentsFiltration(): void {
        this.store.dispatch(GroupsActions.resetAgentsFiltration());
    }

    setAgentsGridSorting(sorting: Sorting) {
        this.store.dispatch(GroupsActions.setAgentsGridSorting({ sorting }));
    }

    selectAgent(id: number): void {
        this.store.dispatch(GroupsActions.selectAgent({ id }));
    }

    selectPolicy(id: number): void {
        this.store.dispatch(GroupsActions.selectPolicy({ id }));
    }

    fetchEvents(id: number, page?: number): void {
        this.store.dispatch(GroupsActions.fetchGroupEvents({ id, page }));
    }

    setEventsGridFiltration(filtration: Filtration): void {
        this.store.dispatch(GroupsActions.setEventsGridFiltration({ filtration }));
    }

    setEventsGridSearch(value: string) {
        this.store.dispatch(GroupsActions.setEventsGridSearch({ value }));
    }

    resetEventsFiltration(): void {
        this.store.dispatch(GroupsActions.resetEventsFiltration());
    }

    setEventsGridSorting(sorting: Sorting) {
        this.store.dispatch(GroupsActions.setEventsGridSorting({ sorting }));
    }

    fetchPolicies(id: number, page?: number): void {
        this.store.dispatch(GroupsActions.fetchGroupPolicies({ id, page }));
    }

    fetchAgentFilterItems() {
        this.store.dispatch(GroupsActions.fetchAgentFilterItems());
    }

    fetchPolicyFilterItems() {
        this.store.dispatch(GroupsActions.fetchPolicyFilterItems());
    }

    fetchEventFilterItems() {
        this.store.dispatch(GroupsActions.fetchEventFilterItems());
    }

    setPoliciesGridFiltration(filtration: Filtration): void {
        this.store.dispatch(GroupsActions.setPoliciesGridFiltration({ filtration }));
    }

    setPoliciesGridSearch(value: string): void {
        this.store.dispatch(GroupsActions.setPoliciesGridSearch({ value }));
    }

    setPoliciesGridSorting(sorting: Sorting) {
        this.store.dispatch(GroupsActions.setPoliciesGridSorting({ sorting }));
    }

    resetPoliciesFiltration(): void {
        this.store.dispatch(GroupsActions.resetPoliciesFiltration());
    }

    selectGroup(id: string): void {
        this.store.dispatch(GroupsActions.selectGroup({ id }));
    }

    selectGroups(groups: Group[]) {
        this.store.dispatch(GroupsActions.selectGroups({ groups }));
    }

    createGroup(group: PrivateGroupInfo): void {
        this.store.dispatch(GroupsActions.createGroup({ group }));
    }

    copyGroup(group: Group, redirect: boolean): void {
        this.store.dispatch(GroupsActions.copyGroup({ group, redirect }));
    }

    updateGroup(group: Group): void {
        this.store.dispatch(GroupsActions.updateGroup({ group }));
    }

    deleteGroup(hash: string): void {
        this.store.dispatch(GroupsActions.deleteGroup({ hash }));
    }

    linkGroupToPolicy(hash: string, policy: ModelsPolicy): void {
        this.store.dispatch(GroupsActions.linkGroupToPolicy({ hash, policy }));
    }

    unlinkGroupFromPolicy(hash: string, policy: ModelsPolicy): void {
        this.store.dispatch(GroupsActions.unlinkGroupFromPolicy({ hash, policy }));
    }

    setGridFiltration(filtration: Filtration): void {
        this.store.dispatch(GroupsActions.setGridFiltration({ filtration }));
    }

    setGridFiltrationByTag(tag: string): void {
        this.store.dispatch(GroupsActions.setGridFiltrationByTag({ tag }));
    }

    setGridSearch(value: string): void {
        this.store.dispatch(GroupsActions.setGridSearch({ value }));
    }

    resetFiltration(): void {
        this.store.dispatch(GroupsActions.resetFiltration());
    }

    setGridSorting(sorting: Sorting): void {
        this.store.dispatch(GroupsActions.setGridSorting({ sorting }));
    }

    getIsExistedGroupsByName(name: string, exclude: string[]): Observable<boolean> {
        const query = allListQuery({
            filters: [
                {
                    field: 'name',
                    value: [name]
                }
            ]
        });

        return this.groupsService.fetchList(query).pipe(
            switchMap((response: SuccessResponse<PrivateGroups>) => from(response.data?.groups)),
            filter(
                (group: ModelsGroup) =>
                    !exclude.some((value) => [group.info.name.ru, group.info.name.en].includes(value))
            ),
            toArray(),
            map((groups) => groups.length > 0),
            catchError(() => of(false))
        );
    }

    fetchGroup(hash: string): void {
        this.store.dispatch(GroupsActions.fetchGroup({ hash }));
    }

    updateAgentData(agent: Agent): void {
        this.store.dispatch(GroupsActions.updateAgentData({ agent }));
    }

    upgradeAgents(agents: Agent[], version: string) {
        const upgradingAgents = agents.filter(
            (agent) => agent.auth_status === 'authorized' && agent.version !== version
        );
        this.store.dispatch(GroupsActions.upgradeAgents({ agents: upgradingAgents, version }));
    }

    cancelUpgradeAgent(hash: string, task: AgentUpgradeTask) {
        this.store.dispatch(GroupsActions.cancelUpgradeAgent({ hash, task }));
    }

    fetchFilterItems() {
        this.store.dispatch(GroupsActions.fetchFilterItems());
    }

    restoreState(): void {
        const params = this.activatedRoute.snapshot.queryParams as Record<string, string>;

        const gridFiltration: Filtration[] = params.filtration ? JSON.parse(params.filtration) : [];
        const gridSearch: string = params.search || '';
        const sorting: Sorting = params.sort ? JSON.parse(params.sort) : { prop: 'name', order: 'ascending' };

        this.store.dispatch(GroupsActions.restoreState({ restoredState: { gridFiltration, gridSearch, sorting } }));
    }

    resetCreatedGroup() {
        this.store.dispatch(GroupsActions.resetCreatedGroup());
    }

    resetGroupErrors() {
        this.store.dispatch(GroupsActions.resetGroupErrors());
    }

    reset() {
        this.store.dispatch(GroupsActions.reset());
    }
}
