import { Injectable } from '@angular/core';
import { Store } from '@ngrx/store';
import { combineLatest, filter, map } from 'rxjs';

import { AgentsService } from '@soldr/api';
import { Agent, AgentUpgradeTask } from '@soldr/models';
import { Filtration, GridColumnFilterItem, LanguageService, Sorting } from '@soldr/shared';

import * as AgentCardActions from './agent-card.actions';
import { State } from './agent-card.reducer';
import {
    selectAgentEvents,
    selectAgentModules,
    selectDeleteError,
    selectEventFilterItemModuleIds,
    selectEventsGridFiltration,
    selectEventsGridFiltrationByField,
    selectEventsGridSearch,
    selectEventsPage,
    selectIsBlockingAgent,
    selectIsDeletingAgent,
    selectIsDeletingFromGroup,
    selectIsLoadingAgent,
    selectIsLoadingEvents,
    selectIsMovingAgent,
    selectIsUpdatingAgent,
    selectIsUpgradingAgent,
    selectIsUpgradingCancelAgent,
    selectMoveToGroupError,
    selectSelectedAgent,
    selectTotalEvents,
    selectUpdateError
} from './agent-card.selectors';

@Injectable({
    providedIn: 'root'
})
export class AgentCardFacade {
    agent$ = this.store.select(selectSelectedAgent);
    agentEvents$ = this.store.select(selectAgentEvents);
    agentModules$ = this.store.select(selectAgentModules);
    deleteError$ = this.store.select(selectDeleteError);
    eventsGridFiltration$ = this.store.select(selectEventsGridFiltration);
    eventsGridFiltrationByFields$ = this.store.select(selectEventsGridFiltrationByField);
    eventsPage$ = this.store.select(selectEventsPage);
    eventsSearchValue$ = this.store.select(selectEventsGridSearch);
    isBlockingAgent$ = this.store.select(selectIsBlockingAgent);
    isCancelUpgradingAgent$ = this.store.select(selectIsUpgradingCancelAgent);
    isDeletingAgent$ = this.store.select(selectIsDeletingAgent);
    isDeletingFromGroup$ = this.store.select(selectIsDeletingFromGroup);
    isLoadingAgent$ = this.store.select(selectIsLoadingAgent);
    isLoadingEvents$ = this.store.select(selectIsLoadingEvents);
    isMovingAgent$ = this.store.select(selectIsMovingAgent);
    isUpdatingAgent$ = this.store.select(selectIsUpdatingAgent);
    isUpgradingAgent$ = this.store.select(selectIsUpgradingAgent);
    moveToGroupError$ = this.store.select(selectMoveToGroupError);
    totalEvents$ = this.store.select(selectTotalEvents);
    updateError$ = this.store.select(selectUpdateError);

    eventGridFilterItems$ = combineLatest([this.agentModules$, this.store.select(selectEventFilterItemModuleIds)]).pipe(
        filter(([modules]) => !!modules.length),
        map(([modules, moduleIds]) => ({
            modules: moduleIds?.map((id) => modules.find((module) => module.id === parseInt(id))) || []
        }))
    );
    eventGridColumnFilterItems$ = this.eventGridFilterItems$.pipe(
        map(({ modules }): { [field: string]: GridColumnFilterItem[] } => ({
            modules: modules.map((module) => ({
                label: module?.locale.module[this.languageService.lang].title,
                value: module?.info.name
            }))
        }))
    );

    constructor(
        private agentsService: AgentsService,
        private languageService: LanguageService,
        private store: Store<State>
    ) {}

    fetchAgent(hash: string): void {
        this.store.dispatch(AgentCardActions.fetchAgent({ hash }));
    }

    upgradeAgent(agent: Agent, version: string) {
        const upgradingAgent = agent.auth_status === 'authorized' && agent.version !== version ? agent : undefined;
        this.store.dispatch(AgentCardActions.upgradeAgent({ agent: upgradingAgent, version }));
    }

    cancelUpgradeAgent(hash: string, task: AgentUpgradeTask) {
        this.store.dispatch(AgentCardActions.cancelUpgradeAgent({ hash, task }));
    }

    moveToGroup(id: number, groupId: number): void {
        this.store.dispatch(AgentCardActions.moveAgentToGroup({ id, groupId }));
    }

    moveToNewGroup(id: number, groupName: string): void {
        this.store.dispatch(AgentCardActions.moveAgentToNewGroup({ id, groupName }));
    }

    updateAgent(agent: Agent) {
        this.store.dispatch(AgentCardActions.updateAgent({ agent }));
    }

    blockAgent(id: number) {
        this.store.dispatch(AgentCardActions.blockAgent({ id }));
    }

    deleteAgent(id: number) {
        this.store.dispatch(AgentCardActions.deleteAgent({ id }));
    }

    fetchEvents(id: number, page?: number): void {
        this.store.dispatch(AgentCardActions.fetchAgentEvents({ id, page }));
    }

    fetchEventFilterItems() {
        this.store.dispatch(AgentCardActions.fetchEventFilterItems());
    }

    setEventsGridFiltration(filtration: Filtration): void {
        this.store.dispatch(AgentCardActions.setEventsGridFiltration({ filtration }));
    }

    setEventsGridSearch(value: string): void {
        this.store.dispatch(AgentCardActions.setEventsGridSearch({ value }));
    }

    resetEventsFiltration(): void {
        this.store.dispatch(AgentCardActions.resetEventsFiltration());
    }

    setEventsGridSorting(sorting: Sorting): void {
        this.store.dispatch(AgentCardActions.setEventsGridSorting({ sorting }));
    }

    reset() {
        this.store.dispatch(AgentCardActions.reset());
    }
}
