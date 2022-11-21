import { Injectable } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';
import { shareReplay, switchMap, take } from 'rxjs';

import { GroupsService, PrivateGroups, SuccessResponse } from '@soldr/api';
import { Group, privateGroupsToModels } from '@soldr/models';
import { ExporterService, LanguageService, ModalInfoService, OsFormatterService } from '@soldr/shared';
import { GroupsFacade } from '@soldr/store/groups';

@Injectable({
    providedIn: 'root'
})
export class GroupsExporterService extends ExporterService {
    // eslint-disable @typescript-eslint/naming-convention
    mapper: Record<string, (value: any) => string> = {
        'details.consistency': (group: Group) =>
            group.details.consistency
                ? ''
                : this.transloco.translate('shared.Shared.Consistency.TooltipText.Inconsistency'),
        'info.name': (group: Group) => group.info.name[this.languageService.lang],
        'details.agents': (group: Group) => `${group.details?.agents || 0}`,
        'details.modules': (group: Group) =>
            group.details?.modules?.map((module) => module.locale.module[this.languageService.lang].title).join(', '),
        'details.policies': (group: Group) =>
            group.details?.policies?.map((policy) => policy.info.name[this.languageService.lang]).join(', '),
        'info.tags': (group: Group) => group.info.tags.join(', ')
    };
    headers: Record<string, () => string> = {
        'details.consistency': () => this.transloco.translate('groups.Groups.GroupsList.GridColumnsTitle.Consistency'),
        'info.name': () => this.transloco.translate('groups.Groups.GroupsList.GridColumnsTitle.Name'),
        'details.agents': () => this.transloco.translate('groups.Groups.GroupsList.GridColumnsTitle.AgentsCount'),
        'details.modules': () => this.transloco.translate('groups.Groups.GroupsList.GridColumnsTitle.Modules'),
        'details.policies': () => this.transloco.translate('groups.Groups.GroupsList.GridColumnsTitle.Policies'),
        'info.tags': () => this.transloco.translate('groups.Groups.GroupsList.GridColumnsTitle.Tags'),
        created_date: () => this.transloco.translate('groups.Groups.GroupsList.GridColumnsTitle.Created'),
        hash: () => this.transloco.translate('groups.Groups.GroupsList.GridColumnsTitle.Hash')
    };
    // eslint-enable @typescript-eslint/naming-convention

    constructor(
        private groupsFacade: GroupsFacade,
        private groupsService: GroupsService,
        private exporterService: ExporterService,
        private modalInfoService: ModalInfoService,
        private transloco: TranslocoService,
        private osFormatter: OsFormatterService,
        private languageService: LanguageService
    ) {
        super();
    }

    export(columns: string[], groups?: Group[]) {
        const filename = this.transloco.translate('groups.Groups.ExportGroups.Text.ExportedFileName');
        this.groupsFacade.allListQuery$
            .pipe(
                shareReplay({ bufferSize: 1, refCount: true }),
                take(1),
                switchMap((initialQuery) => {
                    const query =
                        groups?.length > 0
                            ? {
                                  ...initialQuery,
                                  filters: [...initialQuery.filters, { field: 'id', value: groups.map(({ id }) => id) }]
                              }
                            : initialQuery;

                    return this.groupsService.fetchList(query);
                })
            )
            .subscribe({
                next: (response: SuccessResponse<PrivateGroups>) => {
                    const data = privateGroupsToModels(response.data);
                    super.toCsv(filename, data, columns);
                },
                error: () => {
                    this.modalInfoService.openErrorInfoModal(
                        this.transloco.translate('groups.Groups.ExportGroups.ErrorText.Error')
                    );
                }
            });
    }
}
