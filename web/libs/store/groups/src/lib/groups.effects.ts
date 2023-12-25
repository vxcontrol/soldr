import { HttpErrorResponse } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Router } from '@angular/router';
import { TranslocoService } from '@ngneat/transloco';
import { Actions, createEffect, ofType } from '@ngrx/effects';
import { Store } from '@ngrx/store';
import { debounceTime, filter, forkJoin, of, tap, withLatestFrom } from 'rxjs';
import { catchError, map, switchMap } from 'rxjs/operators';

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
    ModelsGroup,
    PoliciesService,
    PoliciesSQLMappers,
    PrivateAgents,
    PrivateEvents,
    PrivateGroup,
    PrivateGroupModules,
    PrivateGroups,
    PrivatePolicies,
    PrivateTags,
    SuccessResponse,
    TagsService,
    UpgradesService
} from '@soldr/api';
import { groupToDto } from '@soldr/models';
import { DEBOUNCING_DURATION_FOR_REQUESTS, ModalInfoService } from '@soldr/shared';
import {
    selectAgentsGridFiltration,
    selectAgentsGridSearch,
    selectGroup,
    selectEventsGridSearch,
    selectPoliciesGridSearch,
    selectPoliciesGridFiltration,
    selectEventsGridFiltration,
    selectAgentsGridSorting,
    selectEventsGridSorting,
    selectInitialListQuery,
    selectGridFiltration,
    selectPoliciesGridSorting
} from '@soldr/store/groups';

import * as GroupsActions from './groups.actions';
import { State } from './groups.reducer';

const GROUP_FILTER_FIELD_ID = 'group_id';

@Injectable()
export class GroupsEffects {
    fetchGroupsPage$ = createEffect(() =>
        this.actions$.pipe(
            ofType(GroupsActions.fetchGroupsPage),
            debounceTime(DEBOUNCING_DURATION_FOR_REQUESTS),
            withLatestFrom(this.store.select(selectInitialListQuery)),
            switchMap(([action, initialQuery]) => {
                const currentPage = action.page || 1;
                const query = defaultListQuery({ ...initialQuery, page: currentPage });

                return this.groupsService.fetchList(query).pipe(
                    map((response: SuccessResponse<PrivateGroups>) =>
                        GroupsActions.fetchGroupsPageSuccess({ data: response.data, page: currentPage })
                    ),
                    catchError(() => of(GroupsActions.fetchGroupsPageFailure()))
                );
            })
        )
    );

    fetchGroup$ = createEffect(() =>
        this.actions$.pipe(
            ofType(GroupsActions.fetchGroup),
            switchMap(({ hash }) =>
                forkJoin([
                    this.groupsService.fetchOne(hash),
                    this.groupsService.fetchModules(hash, allListQuery())
                ]).pipe(
                    map(([groupResponse, modulesResponse]) =>
                        GroupsActions.fetchGroupSuccess({
                            data: (groupResponse as SuccessResponse<PrivateGroup>).data,
                            modules: (modulesResponse as SuccessResponse<PrivateGroupModules>).data
                        })
                    ),
                    catchError(() => of(GroupsActions.fetchGroupFailure()))
                )
            )
        )
    );

    fetchGroupAgents$ = createEffect(() =>
        this.actions$.pipe(
            ofType(GroupsActions.fetchGroupAgents),
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
                                { field: 'group_id', value: [action.id] },
                                ...filtration
                            ],
                            sort: sorting || {}
                        })
                    )
                    .pipe(
                        map((agentsResponse) =>
                            GroupsActions.fetchGroupAgentsSuccess({
                                data: (agentsResponse as SuccessResponse<PrivateAgents>).data,
                                page
                            })
                        ),
                        catchError(() => of(GroupsActions.fetchGroupAgentsFailure()))
                    );
            })
        )
    );

    fetchGroupEvents$ = createEffect(() =>
        this.actions$.pipe(
            ofType(GroupsActions.fetchGroupEvents),
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
                                { field: 'group_id', value: [action.id] },
                                ...filtration
                            ],
                            sort: sorting || {}
                        })
                    )
                    .pipe(
                        map((eventsResponse) =>
                            GroupsActions.fetchGroupEventsSuccess({
                                data: (eventsResponse as SuccessResponse<PrivateEvents>).data,
                                page
                            })
                        ),
                        catchError(() => of(GroupsActions.fetchGroupEventsFailure()))
                    );
            })
        )
    );

    setAgentsGridFiltration$ = createEffect(() =>
        this.actions$.pipe(
            ofType(
                ...[
                    GroupsActions.setAgentsGridFiltration,
                    GroupsActions.setAgentsGridSearch,
                    GroupsActions.setAgentsGridSorting,
                    GroupsActions.resetAgentsFiltration
                ]
            ),
            withLatestFrom(this.store.select(selectGroup)),
            map(([_, group]) => GroupsActions.fetchGroupAgents({ id: group.id, page: 1 }))
        )
    );

    setAgentsGridFilterItems$ = createEffect(() =>
        this.actions$.pipe(
            ofType(...[GroupsActions.setAgentsGridFiltration, GroupsActions.resetAgentsFiltration]),
            map(() => GroupsActions.fetchAgentFilterItems())
        )
    );

    setEventsGridFiltration$ = createEffect(() =>
        this.actions$.pipe(
            ofType(
                ...[
                    GroupsActions.setEventsGridFiltration,
                    GroupsActions.setEventsGridSearch,
                    GroupsActions.setEventsGridSorting,
                    GroupsActions.resetEventsFiltration
                ]
            ),
            withLatestFrom(this.store.select(selectGroup)),
            map(([_, group]) => GroupsActions.fetchGroupEvents({ id: group.id, page: 1 }))
        )
    );

    setEventsGridFilterItems$ = createEffect(() =>
        this.actions$.pipe(
            ofType(...[GroupsActions.setEventsGridFiltration, GroupsActions.resetEventsFiltration]),
            map(() => GroupsActions.fetchEventFilterItems())
        )
    );

    setPoliciesGridFiltration$ = createEffect(() =>
        this.actions$.pipe(
            ofType(
                ...[
                    GroupsActions.setPoliciesGridFiltration,
                    GroupsActions.setPoliciesGridSearch,
                    GroupsActions.setPoliciesGridSorting,
                    GroupsActions.resetPoliciesFiltration
                ]
            ),
            withLatestFrom(this.store.select(selectGroup)),
            map(([_, group]) => GroupsActions.fetchGroupPolicies({ id: group.id, page: 1 }))
        )
    );

    setPoliciesGridFilterItems$ = createEffect(() =>
        this.actions$.pipe(
            ofType(...[GroupsActions.setPoliciesGridFiltration, GroupsActions.resetPoliciesFiltration]),
            map(() => GroupsActions.fetchPolicyFilterItems())
        )
    );

    fetchGroupPolicies$ = createEffect(() =>
        this.actions$.pipe(
            ofType(GroupsActions.fetchGroupPolicies),
            withLatestFrom(
                this.store.select(selectPoliciesGridSearch),
                this.store.select(selectPoliciesGridFiltration),
                this.store.select(selectPoliciesGridSorting)
            ),
            switchMap(([action, search, filtration, sorting]) => {
                const page = action.page || 1;

                return this.policiesService
                    .fetchList(
                        defaultListQuery({
                            page,
                            filters: [
                                { field: 'data', value: search },
                                { field: 'group_id', value: [action.id] },
                                ...filtration
                            ],
                            sort: sorting || {}
                        })
                    )
                    .pipe(
                        map((policiesResponse) =>
                            GroupsActions.fetchGroupPoliciesSuccess({
                                data: (policiesResponse as SuccessResponse<PrivatePolicies>).data,
                                page
                            })
                        ),
                        catchError(() => of(GroupsActions.fetchGroupPoliciesFailure()))
                    );
            })
        )
    );

    setGridFiltration$ = createEffect(() =>
        this.actions$.pipe(
            ofType(
                ...[
                    GroupsActions.setGridFiltration,
                    GroupsActions.setGridFiltrationByTag,
                    GroupsActions.setAgentsGridFiltrationByTag,
                    GroupsActions.setPoliciesGridFiltrationByTag,
                    GroupsActions.setGridSearch,
                    GroupsActions.setGridSorting,
                    GroupsActions.resetFiltration
                ]
            ),
            debounceTime(DEBOUNCING_DURATION_FOR_REQUESTS),
            switchMap(() => [GroupsActions.fetchGroupsPage({ page: 1 }), GroupsActions.fetchFilterItems()])
        )
    );

    createGroup$ = createEffect(() =>
        this.actions$.pipe(
            ofType(GroupsActions.createGroup),
            switchMap(({ group }) =>
                this.groupsService.create(group).pipe(
                    map((response: SuccessResponse<ModelsGroup>) =>
                        GroupsActions.createGroupSuccess({ group: response.data })
                    ),
                    catchError(({ error }: HttpErrorResponse) => of(GroupsActions.createGroupFailure({ error })))
                )
            )
        )
    );

    updateGroup$ = createEffect(() =>
        this.actions$.pipe(
            ofType(GroupsActions.updateGroup),
            switchMap(({ group }) =>
                this.groupsService.update(group.hash, groupToDto(group)).pipe(
                    map(() => GroupsActions.updateGroupSuccess()),
                    catchError(({ error }: HttpErrorResponse) => of(GroupsActions.updateGroupFailure({ error })))
                )
            )
        )
    );

    copyGroup$ = createEffect(() =>
        this.actions$.pipe(
            ofType(GroupsActions.copyGroup),
            switchMap(({ group, redirect }) =>
                this.groupsService
                    .create({
                        name: group.info.name.ru,
                        tags: group.info.tags,
                        from: group.id
                    })
                    .pipe(
                        tap((response: SuccessResponse<ModelsGroup>) => {
                            if (redirect) {
                                this.router.navigate(['/groups', response.data?.hash]);
                            }
                        }),
                        map((response: SuccessResponse<ModelsGroup>) =>
                            GroupsActions.copyGroupSuccess({ group: response.data })
                        ),
                        catchError(({ error }: HttpErrorResponse) => of(GroupsActions.copyGroupFailure({ error })))
                    )
            )
        )
    );

    deleteGroup$ = createEffect(() =>
        this.actions$.pipe(
            ofType(GroupsActions.deleteGroup),
            switchMap(({ hash }) =>
                this.groupsService.delete(hash).pipe(
                    switchMap(() => [GroupsActions.deleteGroupSuccess(), GroupsActions.fetchGroupsPage({ page: 1 })]),
                    catchError(({ error }: HttpErrorResponse) => of(GroupsActions.deleteGroupFailure({ error })))
                )
            )
        )
    );

    linkGroup$ = createEffect(() =>
        this.actions$.pipe(
            ofType(GroupsActions.linkGroupToPolicy),
            switchMap(({ hash, policy }) =>
                this.groupsService.updatePolicy(hash, { action: 'activate', policy }).pipe(
                    map(() => GroupsActions.linkGroupToPolicySuccess()),
                    catchError(({ error }: HttpErrorResponse) => of(GroupsActions.linkGroupToPolicyFailure({ error })))
                )
            )
        )
    );

    unlinkGroup$ = createEffect(() =>
        this.actions$.pipe(
            ofType(GroupsActions.unlinkGroupFromPolicy),
            switchMap(({ hash, policy }) =>
                this.groupsService.updatePolicy(hash, { action: 'deactivate', policy }).pipe(
                    map(() => GroupsActions.unlinkGroupFromPolicySuccess()),
                    catchError(({ error }: HttpErrorResponse) =>
                        of(GroupsActions.unlinkGroupFromPolicyFailure({ error }))
                    )
                )
            )
        )
    );

    selectGroup$ = createEffect(() =>
        this.actions$.pipe(
            ofType(GroupsActions.selectGroups),
            filter(({ groups }) => groups.length === 1),
            map(({ groups }) =>
                GroupsActions.fetchGroup({
                    hash: groups[0].hash
                })
            )
        )
    );

    updateAgentsData$ = createEffect(() =>
        this.actions$.pipe(
            ofType(GroupsActions.updateAgentData),
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
                            GroupsActions.updateAgentDataSuccess({
                                data: (agentsResponse as SuccessResponse<PrivateAgents>).data
                            })
                        ),
                        catchError(() => of(GroupsActions.updateAgentDataFailure()))
                    )
            )
        )
    );

    upgradeAgents$ = createEffect(() =>
        this.actions$.pipe(
            ofType(GroupsActions.upgradeAgents),
            switchMap(({ version, agents }) =>
                this.upgradesService
                    .upgradeAgent({
                        filters: [{ field: 'id', value: agents.map((agent) => agent.id) }],
                        version
                    })
                    .pipe(
                        map(() => GroupsActions.upgradeAgentsSuccess()),
                        catchError(({ error }: HttpErrorResponse) => {
                            this.modalInfoService.openErrorInfoModal(
                                this.transloco.translate('agents.Agents.EditAgent.ErrorText.UpgradeAgent')
                            );

                            return of(GroupsActions.upgradeAgentsFailure({ error }));
                        })
                    )
            )
        )
    );

    cancelUpgradeAgent$ = createEffect(() =>
        this.actions$.pipe(
            ofType(GroupsActions.cancelUpgradeAgent),
            switchMap(({ hash, task }) =>
                this.upgradesService
                    .updateLastAgentDetails(hash, {
                        ...task,
                        status: 'failed',
                        reason: 'Canceled.By.User'
                    })
                    .pipe(
                        map(() => GroupsActions.cancelUpgradeAgentSuccess()),
                        catchError(() => of(GroupsActions.cancelUpgradeAgentFailure()))
                    )
            )
        )
    );

    fetchFilterItems$ = createEffect(() =>
        this.actions$.pipe(
            ofType(GroupsActions.fetchFilterItems),
            withLatestFrom(this.store.select(selectGridFiltration)),
            switchMap(([_, filter]) =>
                forkJoin([
                    this.groupsService.fetchList(allGroupedListQuery({ filters: filter }, GroupsSQLMappers.PolicyId)),
                    this.groupsService.fetchList(allGroupedListQuery({ filters: filter }, GroupsSQLMappers.ModuleName)),
                    this.tagsService.fetchList(
                        allListQuery({ filters: [...filter, { field: 'type', value: 'groups' }] })
                    )
                ]).pipe(
                    map(
                        ([policiesResponse, modulesResponse, tagsResponse]: [
                            SuccessResponse<GroupedData>,
                            SuccessResponse<GroupedData>,
                            SuccessResponse<PrivateTags>
                        ]) =>
                            GroupsActions.fetchFilterItemsSuccess({
                                policyIds: policiesResponse.data.grouped || [],
                                moduleNames: modulesResponse.data.grouped || [],
                                tags: tagsResponse.data.tags || []
                            })
                    ),
                    catchError((error: ErrorResponse) => of(GroupsActions.fetchFilterItemsFailure({ error })))
                )
            )
        )
    );

    fetchAgentFilterItems$ = createEffect(() =>
        this.actions$.pipe(
            ofType(GroupsActions.fetchAgentFilterItems),
            withLatestFrom(this.store.select(selectAgentsGridFiltration), this.store.select(selectGroup)),
            switchMap(([_, filters, { id }]) =>
                forkJoin([
                    this.agentsService.fetchList(
                        allGroupedListQuery(
                            { filters: [...filters, { field: GROUP_FILTER_FIELD_ID, value: [id] }] },
                            AgentsSQLMappers.ModuleName
                        )
                    ),
                    this.agentsService.fetchList(
                        allGroupedListQuery(
                            { filters: [...filters, { field: GROUP_FILTER_FIELD_ID, value: [id] }] },
                            AgentsSQLMappers.GroupId
                        )
                    ),
                    this.agentsService.fetchList(
                        allGroupedListQuery(
                            { filters: [...filters, { field: GROUP_FILTER_FIELD_ID, value: [id] }] },
                            AgentsSQLMappers.Os
                        )
                    ),
                    this.tagsService.fetchList(
                        allListQuery({
                            filters: [
                                ...filters,
                                { field: GROUP_FILTER_FIELD_ID, value: [id] },
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
                            GroupsActions.fetchAgentFilterItemsSuccess({
                                groupIds: groupsResponse.data.grouped,
                                moduleNames: modulesResponse.data.grouped,
                                os: osResponse.data.grouped,
                                tags: tagsResponse.data.tags
                            })
                    ),
                    catchError((error: ErrorResponse) => of(GroupsActions.fetchAgentFilterItemsFailure({ error })))
                )
            )
        )
    );

    fetchPolicyFilterItems$ = createEffect(() =>
        this.actions$.pipe(
            ofType(GroupsActions.fetchPolicyFilterItems),
            withLatestFrom(this.store.select(selectPoliciesGridFiltration), this.store.select(selectGroup)),
            switchMap(([_, filters, { id }]) =>
                forkJoin([
                    this.policiesService.fetchList(
                        allGroupedListQuery(
                            { filters: [...filters, { field: GROUP_FILTER_FIELD_ID, value: [id] }] },
                            PoliciesSQLMappers.ModuleName
                        )
                    ),
                    this.tagsService.fetchList(
                        allListQuery({
                            filters: [
                                ...filters,
                                { field: GROUP_FILTER_FIELD_ID, value: [id] },
                                { field: 'type', value: 'policies' }
                            ]
                        })
                    )
                ]).pipe(
                    map(
                        ([modulesResponse, tagsResponse]: [
                            SuccessResponse<GroupedData>,
                            SuccessResponse<PrivateTags>
                        ]) =>
                            GroupsActions.fetchPolicyFilterItemsSuccess({
                                moduleNames: modulesResponse.data.grouped,
                                tags: tagsResponse.data.tags
                            })
                    ),
                    catchError((error: ErrorResponse) => of(GroupsActions.fetchPolicyFilterItemsFailure({ error })))
                )
            )
        )
    );

    fetchEventFilterItems$ = createEffect(() =>
        this.actions$.pipe(
            ofType(GroupsActions.fetchEventFilterItems),
            withLatestFrom(this.store.select(selectEventsGridFiltration), this.store.select(selectGroup)),
            switchMap(([_, filters, { id }]) =>
                forkJoin([
                    this.eventsService.fetchEvents(
                        allGroupedListQuery(
                            { filters: [...filters, { field: GROUP_FILTER_FIELD_ID, value: [id] }] },
                            EventsSQLMappers.ModuleId
                        )
                    ),
                    this.eventsService.fetchEvents(
                        allGroupedListQuery(
                            { filters: [...filters, { field: GROUP_FILTER_FIELD_ID, value: [id] }] },
                            EventsSQLMappers.AgentName
                        )
                    ),
                    this.eventsService.fetchEvents(
                        allGroupedListQuery(
                            { filters: [...filters, { field: GROUP_FILTER_FIELD_ID, value: [id] }] },
                            EventsSQLMappers.PolicyId
                        )
                    )
                ]).pipe(
                    map(
                        ([modulesResponse, agentsResponse, policiesResponse]: [
                            SuccessResponse<GroupedData>,
                            SuccessResponse<GroupedData>,
                            SuccessResponse<GroupedData>
                        ]) =>
                            GroupsActions.fetchEventFilterItemsSuccess({
                                moduleIds: modulesResponse.data.grouped,
                                agentNames: agentsResponse.data.grouped,
                                policyIds: policiesResponse.data.grouped
                            })
                    ),
                    catchError((error: ErrorResponse) => of(GroupsActions.fetchEventFilterItemsFailure({ error })))
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
        private store: Store<State>,
        private tagsService: TagsService,
        private transloco: TranslocoService,
        private upgradesService: UpgradesService
    ) {}
}
