import { HttpErrorResponse } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';
import { Actions, createEffect, ofType } from '@ngrx/effects';
import { Store } from '@ngrx/store';
import { catchError, forkJoin, map, of, startWith, switchMap, withLatestFrom } from 'rxjs';

import {
    AgentAction,
    AgentsService,
    allGroupedListQuery,
    allListQuery,
    defaultListQuery,
    ErrorResponse,
    EventsService,
    EventsSQLMappers,
    GroupedData,
    GroupsService,
    ModelsGroup,
    ModulesService,
    PrivateAgent,
    PrivateAgentModules,
    PrivateEvents,
    SuccessResponse,
    UpgradesService
} from '@soldr/api';
import { agentToDto } from '@soldr/models';
import { ModalInfoService } from '@soldr/shared';

import * as AgentCardActions from './agent-card.actions';
import { State } from './agent-card.reducer';
import {
    selectEventsGridFiltration,
    selectEventsGridSearch,
    selectEventsGridSorting,
    selectSelectedAgent
} from './agent-card.selectors';

const AGENT_FILTER_FIELD_ID = 'agent_id';

@Injectable()
export class AgentCardEffects {
    fetchAgent$ = createEffect(() =>
        this.actions$.pipe(
            ofType(AgentCardActions.fetchAgent),
            switchMap(({ hash }) =>
                forkJoin([
                    this.agentsService.fetchOne(hash),
                    this.agentsService.fetchModules(hash, allListQuery())
                ]).pipe(
                    map(([agentResponse, modulesResponse]) =>
                        AgentCardActions.fetchAgentSuccess({
                            data: (agentResponse as SuccessResponse<PrivateAgent>).data,
                            modules: (modulesResponse as SuccessResponse<PrivateAgentModules>).data
                        })
                    ),
                    catchError(() => of(AgentCardActions.fetchAgentFailure()))
                )
            )
        )
    );

    fetchEvents$ = createEffect(() =>
        this.actions$.pipe(
            ofType(AgentCardActions.fetchAgentEvents),
            withLatestFrom(
                this.store.select(selectEventsGridSearch),
                this.store.select(selectEventsGridFiltration),
                this.store.select(selectEventsGridSorting)
            ),
            switchMap(([action, search, filtration, sorting]) => {
                const page = action.page || 1;

                return this.eventsService
                    .fetchEvents(
                        defaultListQuery({
                            page,
                            filters: [
                                { field: 'data', value: search },
                                { field: 'agent_id', value: [action.id] },
                                ...filtration
                            ],
                            sort: sorting || {}
                        })
                    )
                    .pipe(
                        map((eventsResponse) =>
                            AgentCardActions.fetchAgentEventsSuccess({
                                data: (eventsResponse as SuccessResponse<PrivateEvents>).data,
                                page
                            })
                        ),
                        catchError(() => of(AgentCardActions.fetchAgentEventsFailure()))
                    );
            })
        )
    );

    fetchEventFilterItems$ = createEffect(() =>
        this.actions$.pipe(
            ofType(AgentCardActions.fetchEventFilterItems),
            withLatestFrom(this.store.select(selectEventsGridFiltration), this.store.select(selectSelectedAgent)),
            switchMap(([_, filters, { id }]) =>
                this.eventsService
                    .fetchEvents(
                        allGroupedListQuery(
                            { filters: [...filters, { field: AGENT_FILTER_FIELD_ID, value: [id] }] },
                            EventsSQLMappers.ModuleId
                        )
                    )
                    .pipe(
                        map((response: SuccessResponse<GroupedData>) =>
                            AgentCardActions.fetchEventFilterItemsSuccess({
                                moduleIds: response.data.grouped
                            })
                        ),
                        catchError((error: ErrorResponse) =>
                            of(AgentCardActions.fetchEventFilterItemsFailure({ error }))
                        )
                    )
            )
        )
    );

    setEventsGridFiltration$ = createEffect(() =>
        this.actions$.pipe(
            ofType(
                ...[
                    AgentCardActions.setEventsGridFiltration,
                    AgentCardActions.setEventsGridSearch,
                    AgentCardActions.setEventsGridSorting,
                    AgentCardActions.resetEventsFiltration
                ]
            ),
            withLatestFrom(this.store.select(selectSelectedAgent)),
            map(([_, agent]) => AgentCardActions.fetchAgentEvents({ id: agent.id, page: 1 }))
        )
    );

    setEventsGridFilterItems$ = createEffect(() =>
        this.actions$.pipe(
            ofType(...[AgentCardActions.setEventsGridFiltration, AgentCardActions.resetEventsFiltration]),
            map(() => AgentCardActions.fetchEventFilterItems())
        )
    );

    upgradeAgent$ = createEffect(() =>
        this.actions$.pipe(
            ofType(AgentCardActions.upgradeAgent),
            switchMap(({ version, agent }) =>
                this.upgradesService
                    .upgradeAgent({
                        filters: [{ field: 'id', value: agent ? [agent.id] : [] }],
                        version
                    })
                    .pipe(
                        map(() => AgentCardActions.upgradeAgentSuccess()),
                        catchError(({ error }: HttpErrorResponse) => {
                            this.modalInfoService.openErrorInfoModal(
                                this.transloco.translate('agents.Agents.EditAgent.ErrorText.UpgradeAgent')
                            );

                            return of(AgentCardActions.upgradeAgentFailure({ error }));
                        })
                    )
            )
        )
    );

    cancelUpgradeAgent$ = createEffect(() =>
        this.actions$.pipe(
            ofType(AgentCardActions.cancelUpgradeAgent),
            switchMap(({ hash, task }) =>
                this.upgradesService
                    .updateLastAgentDetails(hash, {
                        ...task,
                        status: 'failed',
                        reason: 'Canceled.By.User'
                    })
                    .pipe(
                        map(() => AgentCardActions.cancelUpgradeAgentSuccess()),
                        catchError(() => of(AgentCardActions.cancelUpgradeAgentFailure()))
                    )
            )
        )
    );

    moveToNewGroup$ = createEffect(() =>
        this.actions$.pipe(
            ofType(AgentCardActions.moveAgentToNewGroup),
            switchMap(({ id, groupName }) =>
                this.groupsService
                    .create({
                        name: groupName,
                        tags: []
                    })
                    .pipe(map((response: SuccessResponse<ModelsGroup>) => ({ id, groupId: response.data?.id })))
            ),
            switchMap(({ id, groupId }) =>
                this.agentsService.doAction({
                    action: 'move',
                    filters: [{ field: 'id', value: [id] }],
                    to: groupId
                })
            ),
            map(() => AgentCardActions.moveAgentToNewGroupSuccess()),
            catchError(({ error }: HttpErrorResponse, source) =>
                source.pipe(startWith(AgentCardActions.moveAgentToNewGroupFailure({ error })))
            )
        )
    );

    moveToGroup$ = createEffect(() =>
        this.actions$.pipe(
            ofType(AgentCardActions.moveAgentToGroup),
            switchMap(({ id, groupId }) =>
                this.agentsService
                    .doAction({
                        action: 'move',
                        filters: [{ field: 'id', value: [id] }],
                        to: groupId
                    })
                    .pipe(
                        map(() => AgentCardActions.moveAgentToGroupSuccess()),
                        catchError(({ error }: HttpErrorResponse) =>
                            of(AgentCardActions.moveAgentToGroupFailure({ error }))
                        )
                    )
            )
        )
    );

    updateAgent$ = createEffect(() =>
        this.actions$.pipe(
            ofType(AgentCardActions.updateAgent),
            switchMap(({ agent }) =>
                this.agentsService.update(agent.hash, { action: AgentAction.Edit, agent: agentToDto(agent) }).pipe(
                    map(() => AgentCardActions.updateAgentSuccess()),
                    catchError(({ error }: HttpErrorResponse) => of(AgentCardActions.updateAgentFailure({ error })))
                )
            )
        )
    );

    blockAgent$ = createEffect(() =>
        this.actions$.pipe(
            ofType(AgentCardActions.blockAgent),
            switchMap(({ id }) =>
                this.agentsService
                    .doAction({
                        action: 'block',
                        filters: [{ field: 'id', value: [id] }]
                    })
                    .pipe(
                        map(() => AgentCardActions.blockAgentSuccess()),
                        catchError(({ error }: HttpErrorResponse) => {
                            this.modalInfoService.openErrorInfoModal(
                                this.transloco.translate('agents.Agents.EditAgent.ErrorText.BlockAgent')
                            );

                            return of(AgentCardActions.blockAgentFailure({ error }));
                        })
                    )
            )
        )
    );

    deleteAgent$ = createEffect(() =>
        this.actions$.pipe(
            ofType(AgentCardActions.deleteAgent),
            switchMap(({ id }) =>
                this.agentsService
                    .doAction({
                        action: 'delete',
                        filters: [{ field: 'id', value: [id] }]
                    })
                    .pipe(
                        map(() => AgentCardActions.deleteAgentSuccess()),
                        catchError(({ error }: HttpErrorResponse) => of(AgentCardActions.deleteAgentFailure({ error })))
                    )
            )
        )
    );

    constructor(
        private actions$: Actions,
        private agentsService: AgentsService,
        private eventsService: EventsService,
        private groupsService: GroupsService,
        private modalInfoService: ModalInfoService,
        private modulesService: ModulesService,
        private store: Store<State>,
        private transloco: TranslocoService,
        private upgradesService: UpgradesService
    ) {}
}
