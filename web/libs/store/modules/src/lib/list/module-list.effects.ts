import { HttpErrorResponse } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Actions, createEffect, ofType } from '@ngrx/effects';
import { Store } from '@ngrx/store';
import { catchError, debounceTime, map, of, switchMap, withLatestFrom } from 'rxjs';

import {
    allListQuery,
    defaultListQuery,
    ModelsModuleS,
    ModulesService,
    PrivateSystemModules,
    PrivateSystemShortModules,
    PrivateTags,
    SuccessResponse,
    TagsService
} from '@soldr/api';

import * as ModuleListActions from './module-list.actions';
import { State } from './module-list.reducer';
import { selectInitialListQuery } from './module-list.selectors';
import { DEBOUNCING_DURATION_FOR_REQUESTS } from '@soldr/shared';

@Injectable()
export class ModuleListEffects {
    fetchModules$ = createEffect(() =>
        this.actions$.pipe(
            ofType(ModuleListActions.fetchModules),
            debounceTime(DEBOUNCING_DURATION_FOR_REQUESTS),
            switchMap(() =>
                this.modulesService.fetchList(allListQuery()).pipe(
                    map((response: SuccessResponse<PrivateSystemModules>) =>
                        ModuleListActions.fetchModulesSuccess({ data: response.data })
                    ),
                    catchError(() => of(ModuleListActions.fetchModulesFailure()))
                )
            )
        )
    );

    fetchModulesPage$ = createEffect(() =>
        this.actions$.pipe(
            ofType(ModuleListActions.fetchModulesPage),
            debounceTime(DEBOUNCING_DURATION_FOR_REQUESTS),
            withLatestFrom(this.store.select(selectInitialListQuery)),
            switchMap(([action, initialQuery]) => {
                const currentPage = action.page || 1;

                const query = defaultListQuery({ ...initialQuery, page: currentPage });

                return this.modulesService.fetchList(query).pipe(
                    map((response: SuccessResponse<PrivateSystemModules>) =>
                        ModuleListActions.fetchModulesPageSuccess({ data: response.data, page: currentPage })
                    ),
                    catchError(() => of(ModuleListActions.fetchModulesPageFailure()))
                );
            })
        )
    );

    fetchModulesTags$ = createEffect(() =>
        this.actions$.pipe(
            ofType(ModuleListActions.fetchModulesTags),
            withLatestFrom(this.store.select(selectInitialListQuery)),
            switchMap(([_, query]) =>
                this.tagsService
                    .fetchList(allListQuery({ filters: [...query.filters, { field: 'type', value: 'modules' }] }))
                    .pipe(
                        map((response: SuccessResponse<PrivateTags>) =>
                            ModuleListActions.fetchModulesTagsSuccess({ tags: response.data?.tags })
                        ),
                        catchError(({ error }: HttpErrorResponse) =>
                            of(ModuleListActions.fetchModulesTagsFailure({ error }))
                        )
                    )
            )
        )
    );

    createModule$ = createEffect(() =>
        this.actions$.pipe(
            ofType(ModuleListActions.createModule),
            switchMap(({ module }) =>
                this.modulesService.create(module).pipe(
                    switchMap((response: SuccessResponse<ModelsModuleS>) => [
                        ModuleListActions.createModuleSuccess({ module: response.data }),
                        ModuleListActions.selectModulesByIds({ ids: [response.data?.id] })
                    ]),
                    catchError(({ error }: HttpErrorResponse) => of(ModuleListActions.createModuleFailure({ error })))
                )
            )
        )
    );

    importModule$ = createEffect(() =>
        this.actions$.pipe(
            ofType(ModuleListActions.importModule),
            switchMap(({ name, version, data }) =>
                this.modulesService.importModule(name, version, data).pipe(
                    map(() => ModuleListActions.importModuleSuccess()),
                    catchError(({ error }: HttpErrorResponse) => of(ModuleListActions.importModuleFailure({ error })))
                )
            )
        )
    );

    setGridFiltration$ = createEffect(() =>
        this.actions$.pipe(
            ofType(
                ...[
                    ModuleListActions.setGridFiltration,
                    ModuleListActions.setGridFiltrationByTag,
                    ModuleListActions.setGridSearch,
                    ModuleListActions.setGridSorting,
                    ModuleListActions.resetFiltration
                ]
            ),
            map(() => ModuleListActions.fetchModulesPage({ page: 1 }))
        )
    );

    setGridFilterItems$ = createEffect(() =>
        this.actions$.pipe(
            ofType(...[ModuleListActions.setGridFiltration, ModuleListActions.resetFiltration]),
            map(() => ModuleListActions.fetchModulesTags())
        )
    );

    deleteModule$ = createEffect(() =>
        this.actions$.pipe(
            ofType(ModuleListActions.deleteModule),
            switchMap(({ name }) =>
                this.modulesService.deleteModule(name).pipe(
                    switchMap(() => [
                        ModuleListActions.deleteModuleSuccess(),
                        ModuleListActions.fetchModulesPage({ page: 1 })
                    ]),
                    catchError(({ error }: HttpErrorResponse) => of(ModuleListActions.deleteModuleFailure({ error })))
                )
            )
        )
    );

    deleteModuleVersion$ = createEffect(() =>
        this.actions$.pipe(
            ofType(ModuleListActions.deleteModuleVersion),
            switchMap(({ name, version }) =>
                this.modulesService.deleteModuleVersion(name, version).pipe(
                    switchMap(() => [
                        ModuleListActions.deleteModuleVersionSuccess(),
                        ModuleListActions.fetchModulesPage({ page: 1 })
                    ]),
                    catchError(({ error }: HttpErrorResponse) =>
                        of(ModuleListActions.deleteModuleVersionFailure({ error }))
                    )
                )
            )
        )
    );

    fetchModuleVersions$ = createEffect(() =>
        this.actions$.pipe(
            ofType(ModuleListActions.fetchModuleVersions),
            switchMap(({ name }) =>
                this.modulesService.fetchVersions(name).pipe(
                    map((response: SuccessResponse<PrivateSystemShortModules>) =>
                        ModuleListActions.fetchModuleVersionsSuccess({ data: response.data })
                    ),
                    catchError(() => of(ModuleListActions.fetchModuleVersionsFailure()))
                )
            )
        )
    );

    constructor(
        private actions$: Actions,
        private modulesService: ModulesService,
        private store: Store<State>,
        private tagsService: TagsService
    ) {}
}
