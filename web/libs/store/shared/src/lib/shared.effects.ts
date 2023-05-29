import { HttpErrorResponse } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Router } from '@angular/router';
import { TranslocoService } from '@ngneat/transloco';
import { Actions, createEffect, ofType } from '@ngrx/effects';
import { Store } from '@ngrx/store';
import { McToastService } from '@ptsecurity/mosaic/toast';
import { catchError, debounceTime, from, map, of, switchMap } from 'rxjs';

import {
    AgentsService,
    allListQuery,
    BinariesService,
    defaultListQuery,
    ErrorResponse,
    GroupsService,
    OptionsService,
    PoliciesService,
    Permission,
    PrivateAgents,
    PrivateBinaries,
    PrivateGroups,
    PrivateOptionsActions,
    PrivateOptionsEvents,
    PrivateOptionsTags,
    PrivatePolicies,
    PrivateServices,
    PublicInfo,
    PublicService,
    ServicesService,
    SuccessResponse,
    PrivateSystemModules,
    ModulesService,
    UserService
} from '@soldr/api';
import { PASSWORD_CHANGE_PAGE } from '@soldr/core';
import { DEBOUNCING_DURATION_FOR_REQUESTS, ModalInfoService, saveFile } from '@soldr/shared';

import * as SharedActions from './shared.actions';
import { State } from './shared.reducer';

@Injectable()
export class SharedEffects {
    changePassword$ = createEffect(() =>
        this.actions$.pipe(
            ofType(SharedActions.changePassword),
            switchMap(({ data }) =>
                this.userService.changePassword(data).pipe(
                    map(() => {
                        this.toastService.show({
                            style: 'success',
                            title: this.transloco.translate('shared.Shared.Pseudo.ToastText.SuccessPasswordChange')
                        });

                        return SharedActions.changePasswordSuccess();
                    }),
                    catchError(({ error }: HttpErrorResponse) => of(SharedActions.changePasswordFailure({ error })))
                )
            )
        )
    );

    fetchAllGroups$ = createEffect(() =>
        this.actions$.pipe(
            ofType(SharedActions.fetchAllGroups),
            debounceTime(DEBOUNCING_DURATION_FOR_REQUESTS),
            switchMap(() =>
                this.groupsService.fetchList(allListQuery({ sort: { prop: 'name', order: 'ascending' } })).pipe(
                    map((response: SuccessResponse<PrivateGroups>) =>
                        SharedActions.fetchAllGroupsSuccess({ data: response.data })
                    ),
                    catchError(() => of(SharedActions.fetchAllGroupsFailure()))
                )
            )
        )
    );

    fetchAllAgents$ = createEffect(() =>
        this.actions$.pipe(
            ofType(SharedActions.fetchAllAgents),
            debounceTime(DEBOUNCING_DURATION_FOR_REQUESTS),
            switchMap(() =>
                this.agentsService.fetchList(allListQuery()).pipe(
                    map((response: SuccessResponse<PrivateAgents>) =>
                        SharedActions.fetchAllAgentsSuccess({ data: response.data })
                    ),
                    catchError(() => of(SharedActions.fetchAllAgentsFailure()))
                )
            )
        )
    );

    fetchAllPolicies$ = createEffect(() =>
        this.actions$.pipe(
            ofType(SharedActions.fetchAllPolicies),
            debounceTime(DEBOUNCING_DURATION_FOR_REQUESTS),
            switchMap(() =>
                this.policiesService.fetchList(allListQuery()).pipe(
                    map((response: SuccessResponse<PrivatePolicies>) =>
                        SharedActions.fetchAllPoliciesSuccess({ data: response.data })
                    ),
                    catchError(() => of(SharedActions.fetchAllPoliciesFailure()))
                )
            )
        )
    );

    fetchAllServices$ = createEffect(() =>
        this.actions$.pipe(
            ofType(SharedActions.fetchAllServices),
            debounceTime(DEBOUNCING_DURATION_FOR_REQUESTS),
            switchMap(() =>
                this.servicesService.fetchServiceList(allListQuery()).pipe(
                    map((response: SuccessResponse<PrivateServices>) =>
                        SharedActions.fetchAllServicesSuccess({ data: response.data })
                    ),
                    catchError((error: ErrorResponse) => of(SharedActions.fetchAllServicesFailure({ error })))
                )
            )
        )
    );

    fetchAllModules$ = createEffect(() =>
        this.actions$.pipe(
            ofType(SharedActions.fetchAllModules),
            debounceTime(DEBOUNCING_DURATION_FOR_REQUESTS),
            switchMap(() =>
                this.modulesService.fetchList(allListQuery()).pipe(
                    map((response: SuccessResponse<PrivateSystemModules>) =>
                        SharedActions.fetchAllModulesSuccess({ data: response.data })
                    ),
                    catchError(() => of(SharedActions.fetchAllModulesFailure()))
                )
            )
        )
    );

    exportBinaryFile$ = createEffect(() =>
        this.actions$.pipe(
            ofType(SharedActions.exportBinaryFile),
            switchMap(({ os, arch, pack, version }) =>
                this.binariesService.getBinaryFile(os, arch, pack, version).pipe(
                    map((response) => {
                        saveFile(response);

                        return SharedActions.exportBinaryFileSuccess();
                    }),
                    catchError(({ error }: HttpErrorResponse) => of(SharedActions.exportBinaryFileFailure({ error })))
                )
            )
        )
    );

    fetchInfo$ = createEffect(() =>
        this.actions$.pipe(
            ofType(SharedActions.fetchInfo),
            debounceTime(DEBOUNCING_DURATION_FOR_REQUESTS),
            switchMap(({ refreshCookie }) =>
                this.publicService.getUserInfo(refreshCookie).pipe(
                    map((response: SuccessResponse<PublicInfo>) => {
                        const data = response?.data;
                        if (!data?.privileges?.includes(Permission.ViewServices) && data?.service) {
                            const info = {
                                ...data,
                                service: {
                                    ...data?.service,
                                    name: data?.service?.hash
                                },
                                services: data?.services?.map((service) => ({ ...service, name: service.hash }))
                            };

                            return SharedActions.fetchInfoSuccess({ info });
                        }

                        return SharedActions.fetchInfoSuccess({ info: data });
                    })
                )
            )
        )
    );

    fetchLatestVersion$ = createEffect(() =>
        this.actions$.pipe(
            ofType(SharedActions.fetchLatestAgentBinary),
            switchMap(() =>
                this.binariesService
                    .getBinaries({
                        ...defaultListQuery({
                            sort: {
                                prop: 'version',
                                order: 'descending'
                            },
                            page: 1,
                            pageSize: 1
                        })
                    })
                    .pipe(
                        map((response: SuccessResponse<PrivateBinaries>) =>
                            SharedActions.fetchLatestAgentBinarySuccess({
                                binaries: response?.data
                            })
                        ),
                        catchError(() => of(SharedActions.fetchLatestAgentBinaryFailure()))
                    )
            )
        )
    );

    fetchVersions$ = createEffect(() =>
        this.actions$.pipe(
            ofType(SharedActions.fetchAgentBinaries),
            debounceTime(DEBOUNCING_DURATION_FOR_REQUESTS),
            switchMap(() =>
                this.binariesService
                    .getBinaries({
                        ...defaultListQuery({
                            sort: {
                                prop: 'version',
                                order: 'descending'
                            },
                            page: 1,
                            pageSize: -1
                        })
                    })
                    .pipe(
                        map((response: SuccessResponse<PrivateBinaries>) =>
                            SharedActions.fetchAgentBinariesSuccess({
                                binaries: response?.data
                            })
                        ),
                        catchError(() => of(SharedActions.fetchAgentBinariesFailure()))
                    )
            )
        )
    );

    logout$ = createEffect(() =>
        this.actions$.pipe(
            ofType(SharedActions.logout),
            switchMap(() => this.publicService.logout().pipe(catchError(() => of(SharedActions.logoutFailure())))),
            switchMap(() =>
                from(
                    this.router.navigate(['/login'], {
                        queryParams: { nextUrl: window.location.pathname.replace(PASSWORD_CHANGE_PAGE, '') }
                    })
                ).pipe(
                    map(() => SharedActions.logoutSuccess()),
                    catchError(() => of(SharedActions.logoutFailure()))
                )
            )
        )
    );

    fetchOptionsActions$ = createEffect(() =>
        this.actions$.pipe(
            ofType(SharedActions.fetchOptionsActions),
            debounceTime(DEBOUNCING_DURATION_FOR_REQUESTS),
            switchMap(() =>
                this.optionsService.fetchOptionsActions(allListQuery()).pipe(
                    map((response: SuccessResponse<PrivateOptionsActions>) =>
                        SharedActions.fetchOptionsActionsSuccess({ data: response.data })
                    ),
                    catchError((error: ErrorResponse) => of(SharedActions.fetchOptionsActionsFailure({ error })))
                )
            )
        )
    );

    fetchOptionsEvents$ = createEffect(() =>
        this.actions$.pipe(
            ofType(SharedActions.fetchOptionsEvents),
            debounceTime(DEBOUNCING_DURATION_FOR_REQUESTS),
            switchMap(() =>
                this.optionsService.fetchOptionsEvents(allListQuery()).pipe(
                    map((response: SuccessResponse<PrivateOptionsEvents>) =>
                        SharedActions.fetchOptionsEventsSuccess({ data: response.data })
                    ),
                    catchError((error: ErrorResponse) => of(SharedActions.fetchOptionsEventsFailure({ error })))
                )
            )
        )
    );

    fetchOptionsFields$ = createEffect(() =>
        this.actions$.pipe(
            ofType(SharedActions.fetchOptionsFields),
            debounceTime(DEBOUNCING_DURATION_FOR_REQUESTS),
            switchMap(() =>
                this.optionsService.fetchOptionsFields(allListQuery()).pipe(
                    map((response: SuccessResponse<PrivateOptionsEvents>) =>
                        SharedActions.fetchOptionsFieldsSuccess({ data: response.data })
                    ),
                    catchError((error: ErrorResponse) => of(SharedActions.fetchOptionsFieldsFailure({ error })))
                )
            )
        )
    );

    fetchOptionsTags$ = createEffect(() =>
        this.actions$.pipe(
            ofType(SharedActions.fetchOptionsTags),
            debounceTime(DEBOUNCING_DURATION_FOR_REQUESTS),
            switchMap(() =>
                this.optionsService.fetchOptionsTags(allListQuery()).pipe(
                    map((response: SuccessResponse<PrivateOptionsTags>) =>
                        SharedActions.fetchOptionsTagsSuccess({ data: response.data })
                    ),
                    catchError((error: ErrorResponse) => of(SharedActions.fetchOptionsTagsFailure({ error })))
                )
            )
        )
    );

    constructor(
        private actions$: Actions,
        private agentsService: AgentsService,
        private binariesService: BinariesService,
        private groupsService: GroupsService,
        private modalInfoService: ModalInfoService,
        private modulesService: ModulesService,
        private optionsService: OptionsService,
        private policiesService: PoliciesService,
        private publicService: PublicService,
        private router: Router,
        private servicesService: ServicesService,
        private store: Store<State>,
        private toastService: McToastService,
        private transloco: TranslocoService,
        private userService: UserService
    ) {}
}
