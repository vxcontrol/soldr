import { HttpErrorResponse } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';
import { Actions, createEffect, ofType } from '@ngrx/effects';
import { Store } from '@ngrx/store';
import { catchError, filter, first, forkJoin, map, of, switchMap, withLatestFrom, zip } from 'rxjs';

import {
    AgentAction,
    AgentsService,
    AgentsSQLMappers,
    allGroupedListQuery,
    allListQuery,
    defaultListQuery,
    ErrorResponse,
    GroupedData,
    GroupsService,
    ModelsGroup,
    ModulesService,
    PrivateAgent,
    PrivateAgentModules,
    PrivateAgents,
    PrivateTags,
    PrivateVersions,
    SuccessResponse,
    TagsService,
    UpgradesService,
    VersionsService
} from '@soldr/api';
import { agentToDto } from '@soldr/models';
import { Filter, ModalInfoService } from '@soldr/shared';

import * as AgentListActions from './agent-list.actions';
import { State } from './agent-list.reducer';
import { selectFilters, selectInitialListQuery } from './agent-list.selectors';

@Injectable()
export class AgentListEffects {
    fetchFiltersCounters$ = createEffect(() =>
        this.actions$.pipe(
            ofType(AgentListActions.fetchCountersByFilters),
            switchMap(() =>
                this.store.select(selectFilters).pipe(
                    first(),
                    map((filters: Filter[]) => filters.map((filter) => this.fetchAgentsCountByFilter(filter))),
                    switchMap((v) => zip(...v)),
                    map((counters) =>
                        counters.reduce((acc, { count, id }) => {
                            acc[id] = count;

                            return acc;
                        }, {} as Record<string, number>)
                    ),
                    map((counters) => AgentListActions.fetchCountersByFiltersSuccess({ counters }))
                )
            )
        )
    );

    setGridFiltration$ = createEffect(() =>
        this.actions$.pipe(
            ofType(
                ...[
                    AgentListActions.selectFilter,
                    AgentListActions.selectGroup,
                    AgentListActions.setGridFiltration,
                    AgentListActions.setGridFiltrationByTag,
                    AgentListActions.setGridSearch,
                    AgentListActions.setGridSorting,
                    AgentListActions.resetFiltration
                ]
            ),
            switchMap(() => [AgentListActions.fetchAgentsPage({ page: 1 }), AgentListActions.fetchFilterItems()])
        )
    );

    fetchAgentsPage$ = createEffect(() =>
        this.actions$.pipe(
            ofType(AgentListActions.fetchAgentsPage),
            withLatestFrom(this.store.select(selectInitialListQuery)),
            switchMap(([action, initialQuery]) => {
                const currentPage = action.page || 1;
                const query = defaultListQuery({ ...initialQuery, page: currentPage });

                return this.agentsService.fetchList(query).pipe(
                    map((response: SuccessResponse<PrivateAgents>) =>
                        AgentListActions.fetchAgentsPageSuccess({ data: response.data, page: currentPage })
                    ),
                    catchError(() => of(AgentListActions.fetchAgentsFailure()))
                );
            })
        )
    );

    fetchFilterItems$ = createEffect(() =>
        this.actions$.pipe(
            ofType(AgentListActions.fetchFilterItems),
            withLatestFrom(this.store.select(selectInitialListQuery)),
            filter(([_, filter]) => !!filter),
            switchMap(([_, { filters }]) =>
                forkJoin([
                    this.agentsService.fetchList(allGroupedListQuery({ filters }, AgentsSQLMappers.Version)),
                    this.agentsService.fetchList(allGroupedListQuery({ filters }, AgentsSQLMappers.ModuleName)),
                    this.agentsService.fetchList(allGroupedListQuery({ filters }, AgentsSQLMappers.GroupId)),
                    this.agentsService.fetchList(allGroupedListQuery({ filters }, AgentsSQLMappers.Os)),
                    this.tagsService.fetchList(
                        allListQuery({ filters: [...filters, { field: 'type', value: 'agents' }] })
                    )
                ]).pipe(
                    map(
                        ([versionResponse, modulesResponse, groupsResponse, osResponse, tagsResponse]: [
                            SuccessResponse<GroupedData>,
                            SuccessResponse<GroupedData>,
                            SuccessResponse<GroupedData>,
                            SuccessResponse<GroupedData>,
                            SuccessResponse<PrivateTags>
                        ]) =>
                            AgentListActions.fetchFilterItemsSuccess({
                                groupIds: groupsResponse.data.grouped,
                                moduleNames: modulesResponse.data.grouped,
                                versions: versionResponse.data.grouped,
                                os: osResponse.data.grouped,
                                tags: tagsResponse.data.tags
                            })
                    ),
                    catchError((error: ErrorResponse) => of(AgentListActions.fetchFilterItemsFailure({ error })))
                )
            )
        )
    );

    fetchAgentsVersions$ = createEffect(() =>
        this.actions$.pipe(
            ofType(AgentListActions.fetchAgentsVersions),
            switchMap(() =>
                this.versionService
                    .getVersions('agents')
                    .pipe(
                        map((response: SuccessResponse<PrivateVersions>) =>
                            AgentListActions.fetchAgentsVersionsSuccess({ versions: response.data?.versions })
                        )
                    )
            )
        )
    );

    fetchAgent$ = createEffect(() =>
        this.actions$.pipe(
            ofType(AgentListActions.fetchAgent),
            switchMap(({ hash }) =>
                forkJoin([
                    this.agentsService.fetchOne(hash),
                    this.agentsService.fetchModules(hash, allListQuery())
                ]).pipe(
                    map(([agentResponse, modulesResponse]) =>
                        AgentListActions.fetchAgentSuccess({
                            data: (agentResponse as SuccessResponse<PrivateAgent>).data,
                            modules: (modulesResponse as SuccessResponse<PrivateAgentModules>).data
                        })
                    ),
                    catchError(() => of(AgentListActions.fetchAgentFailure()))
                )
            )
        )
    );

    upgradeAgents$ = createEffect(() =>
        this.actions$.pipe(
            ofType(AgentListActions.upgradeAgents),
            switchMap(({ version, agents }) =>
                this.upgradesService
                    .upgradeAgent({
                        filters: [{ field: 'id', value: agents.map((agent) => agent.id) }],
                        version
                    })
                    .pipe(
                        map(() => AgentListActions.upgradeAgentsSuccess()),
                        catchError(({ error }: HttpErrorResponse) => {
                            this.modalInfoService.openErrorInfoModal(
                                this.transloco.translate('agents.Agents.EditAgent.ErrorText.UpgradeAgent')
                            );

                            return of(AgentListActions.upgradeAgentsFailure({ error }));
                        })
                    )
            )
        )
    );

    cancelUpgradeAgent$ = createEffect(() =>
        this.actions$.pipe(
            ofType(AgentListActions.cancelUpgradeAgent),
            switchMap(({ hash, task }) =>
                this.upgradesService
                    .updateLastAgentDetails(hash, {
                        ...task,
                        status: 'failed',
                        reason: 'Canceled.By.User'
                    })
                    .pipe(
                        map(() => AgentListActions.cancelUpgradeAgentSuccess()),
                        catchError(() => of(AgentListActions.cancelUpgradeAgentFailure()))
                    )
            )
        )
    );

    moveToNewGroup$ = createEffect(() =>
        this.actions$.pipe(
            ofType(AgentListActions.moveAgentsToNewGroup),
            switchMap(({ ids, groupName }) =>
                this.groupsService
                    .create({
                        name: groupName,
                        tags: []
                    })
                    .pipe(map((response: SuccessResponse<ModelsGroup>) => ({ ids, groupId: response.data?.id })))
            ),
            switchMap(({ ids, groupId }) =>
                this.agentsService.doAction({
                    action: 'move',
                    filters: [{ field: 'id', value: ids }],
                    to: groupId
                })
            ),
            map(() => AgentListActions.moveAgentsToNewGroupSuccess()),
            catchError(({ error }: HttpErrorResponse) => of(AgentListActions.moveAgentsToNewGroupFailure({ error })))
        )
    );

    moveToGroup$ = createEffect(() =>
        this.actions$.pipe(
            ofType(AgentListActions.moveAgentsToGroup),
            switchMap(({ ids, groupId }) =>
                this.agentsService
                    .doAction({
                        action: 'move',
                        filters: [{ field: 'id', value: ids }],
                        to: groupId
                    })
                    .pipe(
                        map(() => AgentListActions.moveAgentsToGroupSuccess()),
                        catchError(({ error }: HttpErrorResponse) =>
                            of(AgentListActions.moveAgentsToGroupFailure({ error }))
                        )
                    )
            )
        )
    );

    updateAgent$ = createEffect(() =>
        this.actions$.pipe(
            ofType(AgentListActions.updateAgent),
            switchMap(({ agent }) =>
                this.agentsService.update(agent.hash, { action: AgentAction.Edit, agent: agentToDto(agent) }).pipe(
                    map(() => AgentListActions.updateAgentSuccess()),
                    catchError(({ error }: HttpErrorResponse) => of(AgentListActions.updateAgentFailure({ error })))
                )
            )
        )
    );

    blockAgents$ = createEffect(() =>
        this.actions$.pipe(
            ofType(AgentListActions.blockAgents),
            switchMap(({ ids }) =>
                this.agentsService
                    .doAction({
                        action: 'block',
                        filters: [{ field: 'id', value: ids }]
                    })
                    .pipe(
                        switchMap(() => [
                            AgentListActions.blockAgentsSuccess(),
                            AgentListActions.fetchAgentsPage({ page: 1 }),
                            AgentListActions.fetchCountersByFilters()
                        ]),
                        catchError(({ error }: HttpErrorResponse) => {
                            this.modalInfoService.openErrorInfoModal(
                                this.transloco.translate('agents.Agents.EditAgent.ErrorText.BlockAgent')
                            );

                            return of(AgentListActions.blockAgentsFailure({ error }));
                        })
                    )
            )
        )
    );

    deleteAgents$ = createEffect(() =>
        this.actions$.pipe(
            ofType(AgentListActions.deleteAgents),
            switchMap(({ ids }) =>
                this.agentsService
                    .doAction({
                        action: 'delete',
                        filters: [{ field: 'id', value: ids }]
                    })
                    .pipe(
                        switchMap(() => [
                            AgentListActions.deleteAgentsSuccess(),
                            AgentListActions.fetchCountersByFilters()
                        ]),
                        catchError(({ error }: HttpErrorResponse) =>
                            of(AgentListActions.deleteAgentsFailure({ error }))
                        )
                    )
            )
        )
    );

    selectAgent$ = createEffect(() =>
        this.actions$.pipe(
            ofType(AgentListActions.selectAgents),
            filter(({ agents }) => agents.length === 1),
            map(({ agents }) =>
                AgentListActions.fetchAgent({
                    hash: agents[0].hash
                })
            )
        )
    );

    updateAgentsData$ = createEffect(() =>
        this.actions$.pipe(
            ofType(AgentListActions.updateAgentData),
            switchMap(({ agents }) =>
                this.agentsService
                    .fetchList(
                        defaultListQuery({
                            filters: [{ field: 'id', value: agents.map(({ id }) => id) }],
                            sort: {}
                        })
                    )
                    .pipe(
                        map((agentsResponse) =>
                            AgentListActions.updateAgentDataSuccess({
                                data: (agentsResponse as SuccessResponse<PrivateAgents>).data
                            })
                        ),
                        catchError(({ error }: HttpErrorResponse) =>
                            of(AgentListActions.updateAgentDataFailure({ error }))
                        )
                    )
            )
        )
    );

    updateAgentsPage$ = createEffect(() =>
        this.actions$.pipe(
            ofType(AgentListActions.moveAgentsToGroupSuccess, AgentListActions.moveAgentsToNewGroupSuccess),
            map(() => AgentListActions.fetchCountersByFilters())
        )
    );

    private fetchAgentsCountByFilter = (filter: Filter) => {
        const query = allListQuery({ filters: filter.value });

        return this.agentsService.fetchList(query).pipe(
            map((response: SuccessResponse<PrivateAgents>) => ({
                id: filter.id,
                count: response.data.total
            }))
        );
    };

    constructor(
        private actions$: Actions,
        private agentsService: AgentsService,
        private groupsService: GroupsService,
        private modalInfoService: ModalInfoService,
        private modulesService: ModulesService,
        private store: Store<State>,
        private tagsService: TagsService,
        private transloco: TranslocoService,
        private upgradesService: UpgradesService,
        private versionService: VersionsService
    ) {}
}
