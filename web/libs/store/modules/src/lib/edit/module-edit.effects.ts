import { HttpErrorResponse } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';
import { Actions, createEffect, ofType } from '@ngrx/effects';
import { Store } from '@ngrx/store';
import { McToastService } from '@ptsecurity/mosaic/toast';
import * as base64 from 'base64-js';
import { catchError, forkJoin, map, mergeMap, of, switchMap, withLatestFrom } from 'rxjs';

import {
    allListQuery,
    BinariesService,
    ErrorResponse,
    ModelsModuleS,
    ModuleAction,
    ModulesService,
    ModuleState,
    PrivateBinaries,
    PrivatePolicyModulesUpdates,
    PrivateSystemModuleFile,
    PrivateSystemModuleFilePatch,
    PrivateSystemModules,
    PrivateSystemShortModules,
    StatusResponse,
    SuccessResponse
} from '@soldr/api';
import { AgentVersionPipe, ModuleVersionPipe } from '@soldr/shared';

import * as ModuleEditActions from './module-edit.actions';
import { State } from './module-edit.reducer';
import { selectModule } from './module-edit.selectors';

@Injectable()
export class ModuleEditEffects {
    constructor(
        private modulesService: ModulesService,
        private binariesService: BinariesService,
        private toastService: McToastService,
        private actions$: Actions,
        private store: Store<State>,
        private transloco: TranslocoService
    ) {}

    fetchModule$ = createEffect(() =>
        this.actions$.pipe(
            ofType(ModuleEditActions.fetchModule),
            switchMap(({ name, version }) =>
                forkJoin([
                    this.modulesService.fetchOne(name, version),
                    this.modulesService.fetchVersions(name),
                    this.modulesService.fetchUpdates(name, version),
                    this.modulesService.fetchFiles(name, version)
                ]).pipe(
                    map(([moduleResponse, versionResponse, updatesResponse, filesResponse]) =>
                        ModuleEditActions.fetchModuleSuccess({
                            module: (moduleResponse as SuccessResponse<ModelsModuleS>).data,
                            versions: (versionResponse as SuccessResponse<PrivateSystemShortModules>).data?.modules,
                            updates: (updatesResponse as SuccessResponse<PrivatePolicyModulesUpdates>).data,
                            files: (filesResponse as SuccessResponse<string[]>).data
                        })
                    ),
                    catchError(() => of(ModuleEditActions.fetchModuleFailure()))
                )
            )
        )
    );

    fetchModuleVersions$ = createEffect(() =>
        this.actions$.pipe(
            ofType(ModuleEditActions.fetchModuleVersions),
            switchMap(({ name }) =>
                this.modulesService.fetchVersions(name).pipe(
                    map((response: SuccessResponse<PrivateSystemShortModules>) =>
                        ModuleEditActions.fetchModuleVersionsSuccess({ data: response.data })
                    ),
                    catchError(() => of(ModuleEditActions.fetchModuleVersionsFailure()))
                )
            )
        )
    );

    fetchUpdates$ = createEffect(() =>
        this.actions$.pipe(
            ofType(ModuleEditActions.fetchModuleUpdates),
            switchMap(({ name, version }) =>
                this.modulesService.fetchUpdates(name, version).pipe(
                    map((response: SuccessResponse<PrivatePolicyModulesUpdates>) =>
                        ModuleEditActions.fetchModuleUpdatesSuccess({ updates: response.data })
                    ),
                    catchError(() => of(ModuleEditActions.fetchModuleUpdatesFailure()))
                )
            )
        )
    );

    saveModule$ = createEffect(() =>
        this.actions$.pipe(
            ofType(ModuleEditActions.saveModule),
            withLatestFrom(this.store.select(selectModule)),
            switchMap(([, module]) => {
                const name = module.info.name;
                const version = new ModuleVersionPipe().transform(module.info.version);

                return this.modulesService
                    .update(name, version, {
                        action: ModuleAction.Store,
                        module
                    })
                    .pipe(
                        switchMap(() => {
                            this.toastService.show({
                                style: 'success',
                                title: this.transloco.translate('modules.Modules.ModuleEdit.ToastText.SuccessfulSaving')
                            });

                            return [
                                ModuleEditActions.saveModuleSuccess(),
                                ModuleEditActions.fetchModuleUpdates({ name, version })
                            ];
                        }),
                        catchError(({ error }: HttpErrorResponse) => {
                            this.toastService.show({
                                style: 'error',
                                title: this.transloco.translate('modules.Modules.ModuleEdit.ToastText.SaveError')
                            });

                            return of(ModuleEditActions.saveModuleFailure({ error }));
                        })
                    );
            })
        )
    );

    releaseModule$ = createEffect(() =>
        this.actions$.pipe(
            ofType(ModuleEditActions.releaseModule),
            switchMap(({ name, version, module }) =>
                this.modulesService
                    .update(name, version, {
                        action: ModuleAction.Release,
                        module: { ...module, state: ModuleState.Release }
                    })
                    .pipe(
                        switchMap(() => {
                            this.toastService.show({
                                style: 'success',
                                title: this.transloco.translate(
                                    'modules.Modules.ModuleEdit.ToastText.SuccessfulReleaseTitle'
                                ),
                                caption: this.transloco.translate(
                                    'modules.Modules.ModuleEdit.ToastText.SuccessfulReleaseCaption'
                                )
                            });

                            return [
                                ModuleEditActions.releaseModuleSuccess(),
                                ModuleEditActions.fetchModule({ name, version }),
                                ModuleEditActions.fetchModuleVersions({ name })
                            ];
                        }),
                        catchError(({ error }: HttpErrorResponse) =>
                            of(ModuleEditActions.releaseModuleFailure({ error }))
                        )
                    )
            )
        )
    );

    createModuleDraft$ = createEffect(() =>
        this.actions$.pipe(
            ofType(ModuleEditActions.createModuleDraft),
            switchMap(({ name, version, changelog }) =>
                this.modulesService.createDraft(name, version, changelog).pipe(
                    switchMap(() => [
                        ModuleEditActions.createModuleDraftSuccess(),
                        ModuleEditActions.fetchModule({ name, version }),
                        ModuleEditActions.fetchModuleVersions({ name })
                    ]),
                    catchError(({ error }: HttpErrorResponse) =>
                        of(ModuleEditActions.createModuleDraftFailure({ error }))
                    )
                )
            )
        )
    );

    deleteModule$ = createEffect(() =>
        this.actions$.pipe(
            ofType(ModuleEditActions.deleteModule),
            switchMap(({ name }) =>
                this.modulesService.deleteModule(name).pipe(
                    switchMap(() => [ModuleEditActions.deleteModuleSuccess()]),
                    catchError(({ error }: HttpErrorResponse) => of(ModuleEditActions.deleteModuleFailure({ error })))
                )
            )
        )
    );

    deleteModuleVersion$ = createEffect(() =>
        this.actions$.pipe(
            ofType(ModuleEditActions.deleteModuleVersion),
            switchMap(({ name, version }) =>
                this.modulesService.deleteModuleVersion(name, version).pipe(
                    switchMap(() => [ModuleEditActions.deleteModuleVersionSuccess()]),
                    catchError(({ error }: HttpErrorResponse) =>
                        of(ModuleEditActions.deleteModuleVersionFailure({ error }))
                    )
                )
            )
        )
    );

    updateModuleInPolicies$ = createEffect(() =>
        this.actions$.pipe(
            ofType(ModuleEditActions.updateModuleInPolicies),
            switchMap(({ name, version }) =>
                this.modulesService.updateInPolitics(name, version).pipe(
                    switchMap(() => [
                        ModuleEditActions.updateModuleInPoliciesSuccess(),
                        ModuleEditActions.fetchModuleUpdates({ name, version })
                    ]),
                    catchError(({ error }: HttpErrorResponse) =>
                        of(ModuleEditActions.updateModuleInPoliciesFailure({ error }))
                    )
                )
            )
        )
    );

    fetchFiles$ = createEffect(() =>
        this.actions$.pipe(
            ofType(ModuleEditActions.fetchFiles),
            switchMap(({ name, version }) =>
                this.modulesService.fetchFiles(name, version).pipe(
                    map((response: SuccessResponse<string[]>) =>
                        ModuleEditActions.fetchFilesSuccess({ files: response.data })
                    ),
                    catchError(({ error }: HttpErrorResponse) => of(ModuleEditActions.fetchFilesFailure({ error })))
                )
            )
        )
    );

    loadFile$ = createEffect(() =>
        this.actions$.pipe(
            ofType(ModuleEditActions.loadFiles),
            mergeMap(({ moduleName, version, paths }) => {
                const requests = paths.map((path) =>
                    this.modulesService
                        .fetchFile(moduleName, version, path)
                        .pipe(catchError(({ error }: HttpErrorResponse) => of(error)))
                );

                return forkJoin(requests).pipe(
                    switchMap((responses: (SuccessResponse<PrivateSystemModuleFile> | ErrorResponse)[]) => {
                        const errors = responses
                            .filter((response) => response.status === StatusResponse.Error)
                            .map((error, index) => ({
                                path: paths[index],
                                error: error as ErrorResponse
                            }));
                        const files = responses
                            .filter((response) => response.status === StatusResponse.Success)
                            .map((response, index) => ({
                                path: paths[index],
                                content: new TextDecoder('utf-8').decode(
                                    base64.toByteArray(
                                        (response as SuccessResponse<PrivateSystemModuleFile>).data?.data
                                    )
                                )
                            }));

                        return [
                            ModuleEditActions.loadFilesSuccess({ files }),
                            ...(errors.length > 0 ? [ModuleEditActions.loadFilesFailure({ errors })] : [])
                        ];
                    })
                );
            })
        )
    );

    saveFiles$ = createEffect(() =>
        this.actions$.pipe(
            ofType(ModuleEditActions.saveFiles),
            switchMap(({ moduleName, version, files }) => {
                const requests = files.map((file) => {
                    const patch: PrivateSystemModuleFilePatch = {
                        action: 'save',
                        path: file.path,
                        data: base64.fromByteArray(new TextEncoder().encode(file.content))
                    };

                    return this.modulesService
                        .patchFile(moduleName, version, patch)
                        .pipe(catchError(({ error }: HttpErrorResponse) => of(error)));
                });

                return forkJoin(requests).pipe(
                    switchMap((responses: (SuccessResponse<PrivateSystemModuleFile> | ErrorResponse)[]) => {
                        const errors = responses
                            .filter((response) => response.status === StatusResponse.Error)
                            .map((error, index) => ({
                                path: files[index].path,
                                error: error as ErrorResponse
                            }));

                        return [
                            ModuleEditActions.saveFilesSuccess({ files }),
                            ...(errors.length > 0 ? [ModuleEditActions.saveFilesFailure({ errors })] : []),
                            ModuleEditActions.fetchModuleUpdates({ name: moduleName, version })
                        ];
                    })
                );
            })
        )
    );

    fetchAllModules$ = createEffect(() =>
        this.actions$.pipe(
            ofType(ModuleEditActions.fetchAllModules),
            switchMap(() =>
                this.modulesService.fetchList(allListQuery()).pipe(
                    map((response: SuccessResponse<PrivateSystemModules>) =>
                        ModuleEditActions.fetchAllModulesSuccess({ modules: response.data?.modules })
                    ),
                    catchError(({ error }: HttpErrorResponse) =>
                        of(ModuleEditActions.fetchAllModulesFailure({ error }))
                    )
                )
            )
        )
    );

    fetchAgentVersions$ = createEffect(() =>
        this.actions$.pipe(
            ofType(ModuleEditActions.fetchAgentVersions),
            switchMap(() =>
                this.binariesService
                    .getBinaries(
                        allListQuery({
                            sort: {
                                prop: 'version',
                                order: 'descending'
                            }
                        })
                    )
                    .pipe(
                        map((response: SuccessResponse<PrivateBinaries>) =>
                            ModuleEditActions.fetchAgentVersionsSuccess({
                                agentVersions: response.data?.binaries.map(({ version }) =>
                                    new AgentVersionPipe().transform(version)
                                )
                            })
                        ),
                        catchError(({ error }: HttpErrorResponse) =>
                            of(ModuleEditActions.fetchAgentVersionsFailure({ error }))
                        )
                    )
            )
        )
    );

    fetchModuleVersionsByName$ = createEffect(() =>
        this.actions$.pipe(
            ofType(ModuleEditActions.fetchModuleVersionByName),
            mergeMap(({ name }) =>
                this.modulesService.fetchVersions(name).pipe(
                    map((response: SuccessResponse<PrivateSystemShortModules>) =>
                        ModuleEditActions.fetchModuleVersionByNameSuccess({
                            name,
                            versions: response.data?.modules?.map((module) =>
                                new ModuleVersionPipe().transform(module.info.version)
                            )
                        })
                    ),
                    catchError(({ error }: HttpErrorResponse) =>
                        of(ModuleEditActions.fetchModuleVersionByNameFailure({ error }))
                    )
                )
            )
        )
    );
}
