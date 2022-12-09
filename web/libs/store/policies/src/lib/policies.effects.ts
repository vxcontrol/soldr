import { HttpErrorResponse } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Router } from '@angular/router';
import { TranslocoService } from '@ngneat/transloco';
import { Actions, createEffect, ofType } from '@ngrx/effects';
import { Store } from '@ngrx/store';
import { catchError, debounceTime, filter, forkJoin, map, of, switchMap, tap, withLatestFrom } from 'rxjs';

import {
    AgentsService,
    AgentsSQLMappers,
    allGroupedListQuery,
    allListQuery,
    defaultListQuery,
    ErrorResponse,
    EventsService,
    EventsSQLMappers,
    GroupedData,
    GroupsService,
    GroupsSQLMappers,
    ModelsPolicy,
    PoliciesService,
    PoliciesSQLMappers,
    PrivateAgents,
    PrivateEvents,
    PrivateGroups,
    PrivatePolicies,
    PrivatePolicy,
    PrivatePolicyCountResponse,
    PrivatePolicyModules,
    PrivateTags,
    SuccessResponse,
    TagsService,
    UpgradesService
} from '@soldr/api';
import { policyToDto } from '@soldr/models';
import { DEBOUNCING_DURATION_FOR_REQUESTS, ModalInfoService } from '@soldr/shared';
import { SharedFacade } from '@soldr/store/shared';

import * as PoliciesActions from './policies.actions';
import { State } from './policies.reducer';
import {
    selectAgentsGridFiltration,
    selectAgentsGridSearch,
    selectAgentsGridSorting,
    selectEventsGridFiltration,
    selectEventsGridSearch,
    selectEventsGridSorting,
    selectGroupsGridFiltration,
    selectGroupsGridSearch,
    selectGroupsGridSorting,
    selectInitialListQuery,
    selectPolicy
} from './policies.selectors';

const POLICY_FILTER_FIELD_ID = 'policy_id';

@Injectable()
export class PoliciesEffects {
    fetchFiltersCounters$ = createEffect(() =>
        this.actions$.pipe(
            ofType(PoliciesActions.fetchCountersByFilters),
            debounceTime(DEBOUNCING_DURATION_FOR_REQUESTS),
            switchMap(() =>
                this.policiesService
                    .fetchStatistics()
                    .pipe(
                        map((response: SuccessResponse<PrivatePolicyCountResponse>) =>
                            PoliciesActions.fetchCountersByFiltersSuccess({ counters: response.data })
                        )
                    )
            )
        )
    );

    fetchPoliciesPage$ = createEffect(() =>
        this.actions$.pipe(
            ofType(PoliciesActions.fetchPoliciesPage),
            debounceTime(DEBOUNCING_DURATION_FOR_REQUESTS),
            withLatestFrom(this.store.select(selectInitialListQuery)),
            switchMap(([action, initialQuery]) => {
                const currentPage = action.page || 1;

                const query = defaultListQuery({ ...initialQuery, page: currentPage });

                return this.policiesService.fetchList(query).pipe(
                    map((response: SuccessResponse<PrivatePolicies>) =>
                        PoliciesActions.fetchPoliciesPageSuccess({ data: response.data, page: currentPage })
                    ),
                    catchError(() => of(PoliciesActions.fetchPoliciesPageFailure()))
                );
            })
        )
    );

    fetchFilterItems$ = createEffect(() =>
        this.actions$.pipe(
            ofType(PoliciesActions.fetchFilterItems),
            debounceTime(DEBOUNCING_DURATION_FOR_REQUESTS),
            withLatestFrom(this.store.select(selectInitialListQuery)),
            filter(([_, query]) => !!query),
            switchMap(([_, { filters }]) =>
                forkJoin([
                    this.policiesService.fetchList(allGroupedListQuery({ filters }, PoliciesSQLMappers.ModuleName)),
                    this.policiesService.fetchList(allGroupedListQuery({ filters }, PoliciesSQLMappers.GroupId)),
                    this.tagsService.fetchList(
                        allListQuery({ filters: [...filters, { field: 'type', value: 'policies' }] })
                    )
                ]).pipe(
                    map(
                        ([modulesResponse, groupsResponse, tagsResponse]: [
                            SuccessResponse<GroupedData>,
                            SuccessResponse<GroupedData>,
                            SuccessResponse<PrivateTags>
                        ]) =>
                            PoliciesActions.fetchFilterItemsSuccess({
                                moduleNames: modulesResponse.data.grouped,
                                groupIds: groupsResponse.data.grouped,
                                tags: tagsResponse.data.tags
                            })
                    ),
                    catchError((error: ErrorResponse) => of(PoliciesActions.fetchFilterItemsFailure({ error })))
                )
            )
        )
    );

    fetchPolicy$ = createEffect(() =>
        this.actions$.pipe(
            ofType(PoliciesActions.fetchPolicy),
            switchMap(({ hash }) =>
                this.policiesService.fetchOne(hash).pipe(
                    map((policyResponse) =>
                        PoliciesActions.fetchPolicySuccess({
                            data: (policyResponse as SuccessResponse<PrivatePolicy>).data
                        })
                    ),
                    catchError(() => of(PoliciesActions.fetchPolicyFailure()))
                )
            )
        )
    );

    fetchPolicyModules$ = createEffect(() =>
        this.actions$.pipe(
            ofType(PoliciesActions.fetchPolicyModules),
            debounceTime(DEBOUNCING_DURATION_FOR_REQUESTS),
            switchMap(({ hash }) =>
                this.policiesService.fetchModules(hash, allListQuery()).pipe(
                    map((modulesResponse) =>
                        PoliciesActions.fetchPolicyModulesSuccess({
                            data: (modulesResponse as SuccessResponse<PrivatePolicyModules>).data
                        })
                    ),
                    catchError(() => of(PoliciesActions.fetchPolicyModulesFailure()))
                )
            )
        )
    );

    setGridFiltration$ = createEffect(() =>
        this.actions$.pipe(
            ofType(
                ...[
                    PoliciesActions.selectFilter,
                    PoliciesActions.selectGroup,
                    PoliciesActions.setGridFiltration,
                    PoliciesActions.setGridFiltrationByTag,
                    PoliciesActions.setGridSearch,
                    PoliciesActions.setGridSorting,
                    PoliciesActions.resetFiltration
                ]
            ),
            switchMap(() => [PoliciesActions.fetchPoliciesPage({ page: 1 }), PoliciesActions.fetchFilterItems()])
        )
    );

    createPolicy$ = createEffect(() =>
        this.actions$.pipe(
            ofType(PoliciesActions.createPolicy),
            switchMap(({ policy }) =>
                this.policiesService.create(policy).pipe(
                    switchMap((response: SuccessResponse<ModelsPolicy>) => [
                        PoliciesActions.createPolicySuccess({ policy: response.data }),
                        PoliciesActions.selectPoliciesByIds({ ids: [response.data?.id] }),
                        PoliciesActions.fetchCountersByFilters()
                    ]),
                    catchError(({ error }: HttpErrorResponse) => of(PoliciesActions.createPolicyFailure({ error })))
                )
            )
        )
    );

    updatePolicy$ = createEffect(() =>
        this.actions$.pipe(
            ofType(PoliciesActions.updatePolicy),
            switchMap(({ policy }) =>
                this.policiesService.update(policy.hash, policyToDto(policy)).pipe(
                    map(() => PoliciesActions.updatePolicySuccess()),
                    catchError(({ error }: HttpErrorResponse) => of(PoliciesActions.updatePolicyFailure({ error })))
                )
            )
        )
    );

    copyPolicy$ = createEffect(() =>
        this.actions$.pipe(
            ofType(PoliciesActions.copyPolicy),
            switchMap(({ policy, redirect }) =>
                this.policiesService
                    .create({
                        name: policy.info.name.ru,
                        tags: policy.info.tags,
                        from: policy.id
                    })
                    .pipe(
                        tap((response: SuccessResponse<ModelsPolicy>) => {
                            if (redirect) {
                                this.router.navigate(['/policies', response.data?.hash]);
                            }
                        }),
                        switchMap((response: SuccessResponse<ModelsPolicy>) => [
                            PoliciesActions.copyPolicySuccess({ policy: response.data }),
                            PoliciesActions.fetchCountersByFilters()
                        ]),
                        catchError(({ error }: HttpErrorResponse) => of(PoliciesActions.copyPolicyFailure({ error })))
                    )
            )
        )
    );

    deletePolicy$ = createEffect(() =>
        this.actions$.pipe(
            ofType(PoliciesActions.deletePolicy),
            switchMap(({ hash }) =>
                this.policiesService.delete(hash).pipe(
                    switchMap(() => [PoliciesActions.deletePolicySuccess(), PoliciesActions.fetchCountersByFilters()]),
                    catchError(({ error }: HttpErrorResponse) => of(PoliciesActions.deletePolicyFailure({ error })))
                )
            )
        )
    );

    linkPolicy$ = createEffect(() =>
        this.actions$.pipe(
            ofType(PoliciesActions.linkPolicyToGroup),
            switchMap(({ hash, group }) =>
                this.policiesService.updateGroup(hash, { action: 'activate', group }).pipe(
                    map(() => PoliciesActions.linkPolicyToGroupSuccess()),
                    catchError(({ error }: HttpErrorResponse) =>
                        of(PoliciesActions.linkPolicyToGroupFailure({ error }))
                    )
                )
            )
        )
    );

    unlinkPolicy$ = createEffect(() =>
        this.actions$.pipe(
            ofType(PoliciesActions.unlinkPolicyFromGroup),
            switchMap(({ hash, group }) =>
                this.policiesService.updateGroup(hash, { action: 'deactivate', group }).pipe(
                    map(() => PoliciesActions.unlinkPolicyFromGroupSuccess()),
                    catchError(({ error }: HttpErrorResponse) =>
                        of(PoliciesActions.unlinkPolicyFromGroupFailure({ error }))
                    )
                )
            )
        )
    );

    updateLinks$ = createEffect(
        () =>
            this.actions$.pipe(
                ofType(PoliciesActions.unlinkPolicyFromGroupSuccess, PoliciesActions.linkPolicyToGroupSuccess),
                tap(() => this.sharedFacade.fetchAllPolicies(true))
            ),
        { dispatch: false }
    );

    selectPolicy$ = createEffect(() =>
        this.actions$.pipe(
            ofType(PoliciesActions.selectPolicies),
            filter(({ policies }) => policies.length === 1),
            map(({ policies }) =>
                PoliciesActions.fetchPolicy({
                    hash: policies[0].hash
                })
            )
        )
    );

    fetchPolicyEvents$ = createEffect(() =>
        this.actions$.pipe(
            ofType(PoliciesActions.fetchPolicyEvents),
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
                                { field: 'policy_id', value: [action.id] },
                                { field: 'data', value: search },
                                ...filtration
                            ],
                            sort: sorting || {}
                        })
                    )
                    .pipe(
                        map((eventsResponse) =>
                            PoliciesActions.fetchPolicyEventsSuccess({
                                data: (eventsResponse as SuccessResponse<PrivateEvents>).data,
                                page
                            })
                        ),
                        catchError(() => of(PoliciesActions.fetchPolicyEventsFailure()))
                    );
            })
        )
    );

    setEventsGridFiltration$ = createEffect(() =>
        this.actions$.pipe(
            ofType(
                ...[
                    PoliciesActions.setEventsGridFiltration,
                    PoliciesActions.setEventsGridSearch,
                    PoliciesActions.setEventsGridSorting,
                    PoliciesActions.resetEventsFiltration
                ]
            ),
            withLatestFrom(this.store.select(selectPolicy)),
            map(([, policy]) => PoliciesActions.fetchPolicyEvents({ id: policy.id, page: 1 }))
        )
    );

    setEventsGridFilterItems$ = createEffect(() =>
        this.actions$.pipe(
            ofType(...[PoliciesActions.setEventsGridFiltration, PoliciesActions.resetEventsFiltration]),
            map(() => PoliciesActions.fetchEventFilterItems())
        )
    );

    fetchPolicyAgents$ = createEffect(() =>
        this.actions$.pipe(
            ofType(PoliciesActions.fetchPolicyAgents),
            withLatestFrom(
                this.store.select(selectAgentsGridSearch),
                this.store.select(selectAgentsGridFiltration),
                this.store.select(selectAgentsGridSorting)
            ),
            switchMap(([action, search, filtration, sorting]) => {
                const page = action.page || 1;

                return this.agentsService
                    .fetchList(
                        defaultListQuery({
                            page,
                            filters: [
                                { field: 'data', value: search },
                                { field: 'policy_id', value: [action.id] },
                                ...filtration
                            ],
                            sort: sorting || {}
                        })
                    )
                    .pipe(
                        map((agentsResponse) =>
                            PoliciesActions.fetchPolicyAgentsSuccess({
                                data: (agentsResponse as SuccessResponse<PrivateAgents>).data,
                                page
                            })
                        ),
                        catchError(() => of(PoliciesActions.fetchPolicyAgentsFailure()))
                    );
            })
        )
    );

    setAgentsGridFiltration$ = createEffect(() =>
        this.actions$.pipe(
            ofType(
                ...[
                    PoliciesActions.setAgentsGridFiltration,
                    PoliciesActions.setAgentsGridSearch,
                    PoliciesActions.setAgentsGridSorting,
                    PoliciesActions.resetAgentsFiltration
                ]
            ),
            withLatestFrom(this.store.select(selectPolicy)),
            map(([, policy]) => PoliciesActions.fetchPolicyAgents({ id: policy.id, page: 1 }))
        )
    );

    setAgentsGridFilterItems$ = createEffect(() =>
        this.actions$.pipe(
            ofType(...[PoliciesActions.setAgentsGridFiltration, PoliciesActions.resetAgentsFiltration]),
            map(() => PoliciesActions.fetchAgentFilterItems())
        )
    );

    fetchPolicyGroups$ = createEffect(() =>
        this.actions$.pipe(
            ofType(PoliciesActions.fetchPolicyGroups),
            withLatestFrom(
                this.store.select(selectGroupsGridSearch),
                this.store.select(selectGroupsGridFiltration),
                this.store.select(selectGroupsGridSorting)
            ),
            switchMap(([action, search, filtration, sorting]) => {
                const page = action.page || 1;

                return this.groupsService
                    .fetchList(
                        defaultListQuery({
                            page,
                            filters: [
                                { field: 'data', value: search },
                                { field: 'policy_id', value: [action.id] },
                                ...filtration
                            ],
                            sort: sorting || {}
                        })
                    )
                    .pipe(
                        map((groupsResponse) =>
                            PoliciesActions.fetchPolicyGroupsSuccess({
                                data: (groupsResponse as SuccessResponse<PrivateGroups>).data,
                                page
                            })
                        ),
                        catchError(() => of(PoliciesActions.fetchPolicyGroupsFailure()))
                    );
            })
        )
    );

    setGroupsGridFiltration$ = createEffect(() =>
        this.actions$.pipe(
            ofType(
                ...[
                    PoliciesActions.setGroupsGridFiltration,
                    PoliciesActions.setGroupsGridSearch,
                    PoliciesActions.setGroupsGridSorting,
                    PoliciesActions.resetGroupsFiltration
                ]
            ),
            withLatestFrom(this.store.select(selectPolicy)),
            map(([, policy]) => PoliciesActions.fetchPolicyGroups({ id: policy.id, page: 1 }))
        )
    );

    setGroupsGridFilterItems$ = createEffect(() =>
        this.actions$.pipe(
            ofType(...[PoliciesActions.setGroupsGridFiltration, PoliciesActions.resetGroupsFiltration]),
            map(() => PoliciesActions.fetchGroupFilterItems())
        )
    );

    updateAgentsData$ = createEffect(() =>
        this.actions$.pipe(
            ofType(PoliciesActions.updateAgentData),
            switchMap(({ agent }) =>
                this.agentsService
                    .fetchList(
                        defaultListQuery({
                            filters: [{ field: 'id', value: [agent.id] }],
                            pageSize: 1,
                            sort: {}
                        })
                    )
                    .pipe(
                        map((agentsResponse) =>
                            PoliciesActions.updateAgentDataSuccess({
                                data: (agentsResponse as SuccessResponse<PrivateAgents>).data
                            })
                        ),
                        catchError(() => of(PoliciesActions.updateAgentDataFailure()))
                    )
            )
        )
    );

    updatePolicyModules$ = createEffect(() =>
        this.actions$.pipe(
            ofType(PoliciesActions.fetchPolicy),
            map(({ hash }) => PoliciesActions.fetchPolicyModules({ hash }))
        )
    );

    upgradeAgents$ = createEffect(() =>
        this.actions$.pipe(
            ofType(PoliciesActions.upgradeAgents),
            switchMap(({ version, agents }) =>
                this.upgradesService
                    .upgradeAgent({
                        filters: [{ field: 'id', value: agents.map((agent) => agent.id) }],
                        version
                    })
                    .pipe(
                        map(() => PoliciesActions.upgradeAgentsSuccess()),
                        catchError(({ error }: HttpErrorResponse) => {
                            this.modalInfoService.openErrorInfoModal(
                                this.transloco.translate('agents.Agents.EditAgent.ErrorText.UpgradeAgent')
                            );

                            return of(PoliciesActions.upgradeAgentsFailure({ error }));
                        })
                    )
            )
        )
    );

    cancelUpgradeAgent$ = createEffect(() =>
        this.actions$.pipe(
            ofType(PoliciesActions.cancelUpgradeAgent),
            switchMap(({ hash, task }) =>
                this.upgradesService
                    .updateLastAgentDetails(hash, {
                        ...task,
                        status: 'failed',
                        reason: 'Canceled.By.User'
                    })
                    .pipe(
                        map(() => PoliciesActions.cancelUpgradeAgentSuccess()),
                        catchError(() => of(PoliciesActions.cancelUpgradeAgentFailure()))
                    )
            )
        )
    );

    fetchAgentFilterItems$ = createEffect(() =>
        this.actions$.pipe(
            ofType(PoliciesActions.fetchAgentFilterItems),
            withLatestFrom(this.store.select(selectAgentsGridFiltration), this.store.select(selectPolicy)),
            switchMap(([_, filters, { id }]) =>
                forkJoin([
                    this.agentsService.fetchList(
                        allGroupedListQuery(
                            { filters: [...filters, { field: POLICY_FILTER_FIELD_ID, value: [id] }] },
                            AgentsSQLMappers.ModuleName
                        )
                    ),
                    this.agentsService.fetchList(
                        allGroupedListQuery(
                            { filters: [...filters, { field: POLICY_FILTER_FIELD_ID, value: [id] }] },
                            AgentsSQLMappers.GroupId
                        )
                    ),
                    this.agentsService.fetchList(
                        allGroupedListQuery(
                            { filters: [...filters, { field: POLICY_FILTER_FIELD_ID, value: [id] }] },
                            AgentsSQLMappers.Os
                        )
                    ),
                    this.tagsService.fetchList(
                        allListQuery({
                            filters: [
                                ...filters,
                                { field: POLICY_FILTER_FIELD_ID, value: [id] },
                                { field: 'type', value: 'agents' }
                            ]
                        })
                    )
                ]).pipe(
                    map(
                        ([modulesResponse, groupsResponse, osResponse, tagsResponse]: [
                            SuccessResponse<GroupedData>,
                            SuccessResponse<GroupedData>,
                            SuccessResponse<GroupedData>,
                            SuccessResponse<PrivateTags>
                        ]) =>
                            PoliciesActions.fetchAgentFilterItemsSuccess({
                                groupIds: groupsResponse.data.grouped,
                                moduleNames: modulesResponse.data.grouped,
                                os: osResponse.data.grouped,
                                tags: tagsResponse.data.tags
                            })
                    ),
                    catchError((error: ErrorResponse) => of(PoliciesActions.fetchAgentFilterItemsFailure({ error })))
                )
            )
        )
    );

    fetchEventFilterItems$ = createEffect(() =>
        this.actions$.pipe(
            ofType(PoliciesActions.fetchEventFilterItems),
            withLatestFrom(this.store.select(selectEventsGridFiltration), this.store.select(selectPolicy)),
            switchMap(([_, filters, { id }]) =>
                forkJoin([
                    this.eventsService.fetchEvents(
                        allGroupedListQuery(
                            { filters: [...filters, { field: POLICY_FILTER_FIELD_ID, value: [id] }] },
                            EventsSQLMappers.ModuleId
                        )
                    ),
                    this.eventsService.fetchEvents(
                        allGroupedListQuery(
                            { filters: [...filters, { field: POLICY_FILTER_FIELD_ID, value: [id] }] },
                            EventsSQLMappers.AgentId
                        )
                    ),
                    this.eventsService.fetchEvents(
                        allGroupedListQuery(
                            { filters: [...filters, { field: POLICY_FILTER_FIELD_ID, value: [id] }] },
                            EventsSQLMappers.GroupId
                        )
                    )
                ]).pipe(
                    map(
                        ([modulesResponse, agentsResponse, groupsResponse]: [
                            SuccessResponse<GroupedData>,
                            SuccessResponse<GroupedData>,
                            SuccessResponse<GroupedData>
                        ]) =>
                            PoliciesActions.fetchEventFilterItemsSuccess({
                                moduleIds: modulesResponse.data.grouped,
                                agentIds: agentsResponse.data.grouped,
                                groupIds: groupsResponse.data.grouped
                            })
                    ),
                    catchError((error: ErrorResponse) => of(PoliciesActions.fetchEventFilterItemsFailure({ error })))
                )
            )
        )
    );

    fetchGroupFilterItems$ = createEffect(() =>
        this.actions$.pipe(
            ofType(PoliciesActions.fetchGroupFilterItems),
            withLatestFrom(this.store.select(selectGroupsGridFiltration), this.store.select(selectPolicy)),
            switchMap(([_, filters, { id }]) =>
                forkJoin([
                    this.groupsService.fetchList(
                        allGroupedListQuery(
                            { filters: [...filters, { field: POLICY_FILTER_FIELD_ID, value: [id] }] },
                            GroupsSQLMappers.ModuleName
                        )
                    ),
                    this.groupsService.fetchList(
                        allGroupedListQuery(
                            { filters: [...filters, { field: POLICY_FILTER_FIELD_ID, value: [id] }] },
                            GroupsSQLMappers.PolicyId
                        )
                    ),
                    this.tagsService.fetchList(
                        allListQuery({
                            filters: [
                                ...filters,
                                { field: POLICY_FILTER_FIELD_ID, value: [id] },
                                { field: 'type', value: 'groups' }
                            ]
                        })
                    )
                ]).pipe(
                    map(
                        ([modulesResponse, policiesResponse, tagsResponse]: [
                            SuccessResponse<GroupedData>,
                            SuccessResponse<GroupedData>,
                            SuccessResponse<PrivateTags>
                        ]) =>
                            PoliciesActions.fetchGroupFilterItemsSuccess({
                                moduleNames: modulesResponse.data.grouped,
                                policyIds: policiesResponse.data.grouped,
                                tags: tagsResponse.data.tags
                            })
                    ),
                    catchError((error: ErrorResponse) => of(PoliciesActions.fetchGroupFilterItemsFailure({ error })))
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
        private policiesService: PoliciesService,
        private router: Router,
        private sharedFacade: SharedFacade,
        private store: Store<State>,
        private tagsService: TagsService,
        private transloco: TranslocoService,
        private upgradesService: UpgradesService
    ) {}
}
