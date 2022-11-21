import { Injectable } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';
import { shareReplay, switchMap, take } from 'rxjs';

import { ModulesService, PrivateSystemModules, SuccessResponse } from '@soldr/api';
import { manyModulesToModels, Module } from '@soldr/models';
import {
    Architecture,
    ExporterService,
    LanguageService,
    ModalInfoService,
    ModuleVersionPipe,
    OperationSystem,
    OsFormatterService
} from '@soldr/shared';
import { ModuleListFacade } from '@soldr/store/modules';

@Injectable({
    providedIn: 'root'
})
export class ModulesExporterService extends ExporterService {
    // eslint-disable @typescript-eslint/naming-convention
    mapper: Record<string, (value: any) => string> = {
        'info.name': (module: Module) => module.locale.module[this.languageService.lang].title,
        version: (module: Module) => new ModuleVersionPipe().transform(module.info.version),
        os: (module: Module) =>
            Object.keys(module.info.os)
                .map((family) =>
                    this.osFormatter.getText(family as OperationSystem, module.info.os[family] as Architecture[])
                )
                .join(', '),
        description: (module: Module) => module.locale.module[this.languageService.lang].description,
        'info.tags': (module: Module) =>
            module.info.tags.map((tag) => module.locale.tags[tag][this.languageService.lang].title).join(', ')
    };
    headers: Record<string, () => string> = {
        'info.name': () => this.transloco.translate('modules.Modules.ModulesList.GridColumnsTitle.Name'),
        version: () => this.transloco.translate('modules.Modules.ModulesList.GridColumnsTitle.Version'),
        os: () => this.transloco.translate('modules.Modules.ModulesList.GridColumnsTitle.Os'),
        description: () => this.transloco.translate('modules.Modules.ModulesList.GridColumnsTitle.Description'),
        'info.tags': () => this.transloco.translate('shared.Shared.Pseudo.GridColumnsTitle.Tags')
    };
    // eslint-enable @typescript-eslint/naming-convention

    constructor(
        private modulesFacade: ModuleListFacade,
        private modulesService: ModulesService,
        private exporterService: ExporterService,
        private modalInfoService: ModalInfoService,
        private transloco: TranslocoService,
        private osFormatter: OsFormatterService,
        private languageService: LanguageService
    ) {
        super();
    }

    export(columns: string[], modules?: Module[]) {
        const filename = this.transloco.translate('modules.Modules.ExportModules.Text.ExportedFileName');
        this.modulesFacade.allListQuery$
            .pipe(
                shareReplay({ bufferSize: 1, refCount: true }),
                take(1),
                switchMap((initialQuery) => {
                    const query =
                        modules?.length > 0
                            ? {
                                  ...initialQuery,
                                  filters: [
                                      ...initialQuery.filters,
                                      { field: 'name', value: modules.map(({ info }) => info.name) }
                                  ]
                              }
                            : initialQuery;

                    return this.modulesService.fetchList(query);
                })
            )
            .subscribe({
                next: (response: SuccessResponse<PrivateSystemModules>) => {
                    const data = manyModulesToModels(response.data);
                    super.toCsv(filename, data, columns);
                },
                error: () => {
                    this.modalInfoService.openErrorInfoModal(
                        this.transloco.translate('modules.Modules.ExportModules.ErrorText.Error')
                    );
                }
            });
    }
}
