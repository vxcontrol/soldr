import { Injectable } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';
import { shareReplay, switchMap, take } from 'rxjs';

import { PoliciesService, PrivatePolicies, SuccessResponse } from '@soldr/api';
import { Policy, privatePoliciesToModels } from '@soldr/models';
import { ExporterService, LanguageService, ModalInfoService, OperationSystem, OsFormatterService } from '@soldr/shared';
import { PoliciesFacade } from '@soldr/store/policies';

@Injectable({
    providedIn: 'root'
})
export class PoliciesExporterService extends ExporterService {
    // eslint-disable @typescript-eslint/naming-convention
    mapper: Record<string, (value: any) => string> = {
        'details.consistency': (policy: Policy) =>
            policy.details.consistency
                ? ''
                : this.transloco.translate('shared.Shared.Consistency.TooltipText.Inconsistency'),
        'info.name': (policy: Policy) => policy.info.name[this.languageService.lang],
        'details.modules': (policy: Policy) =>
            policy.details?.modules?.map((module) => module.locale.module[this.languageService.lang].title).join(', '),
        os: (policy: Policy) =>
            Object.keys(policy.info.os)
                .map((family) =>
                    this.osFormatter.getText(family as OperationSystem, policy.info.os[family as OperationSystem])
                )
                .join(', '),
        'details.groups': (policy: Policy) =>
            policy.details?.groups?.map((group) => group.info.name[this.languageService.lang]).join(', '),
        'info.tags': (policy: Policy) => policy.info.tags.join(', ')
    };
    headers: Record<string, () => string> = {
        'details.consistency': () =>
            this.transloco.translate('policies.Policies.PoliciesList.GridColumnsTitle.Consistency'),
        'info.name': () => this.transloco.translate('policies.Policies.PoliciesList.GridColumnsTitle.Name'),
        'details.modules': () => this.transloco.translate('policies.Policies.PoliciesList.GridColumnsTitle.Modules'),
        'details.groups': () => this.transloco.translate('policies.Policies.PoliciesList.GridColumnsTitle.Groups'),
        'info.tags': () => this.transloco.translate('policies.Policies.PoliciesList.GridColumnsTitle.Tags'),
        created_date: () => this.transloco.translate('policies.Policies.PoliciesList.GridColumnsTitle.Created'),
        hash: () => this.transloco.translate('policies.Policies.PoliciesList.GridColumnsTitle.Hash'),
        os: () => this.transloco.translate('policies.Policies.PoliciesList.GridColumnsTitle.Os')
    };
    // eslint-enable @typescript-eslint/naming-convention

    constructor(
        private policiesFacade: PoliciesFacade,
        private policiesService: PoliciesService,
        private exporterService: ExporterService,
        private modalInfoService: ModalInfoService,
        private transloco: TranslocoService,
        private osFormatter: OsFormatterService,
        private languageService: LanguageService
    ) {
        super();
    }

    export(columns: string[], policies?: Policy[]) {
        const filename = this.transloco.translate('policies.Policies.ExportPolicies.Text.ExportedFileName');
        this.policiesFacade.allListQuery$
            .pipe(
                shareReplay({ bufferSize: 1, refCount: true }),
                take(1),
                switchMap((initialQuery) => {
                    const query =
                        policies?.length > 0
                            ? {
                                  ...initialQuery,
                                  filters: [
                                      ...initialQuery.filters,
                                      { field: 'id', value: policies.map(({ id }) => id) }
                                  ]
                              }
                            : initialQuery;

                    return this.policiesService.fetchList(query);
                })
            )
            .subscribe({
                next: (response: SuccessResponse<PrivatePolicies>) => {
                    const data = privatePoliciesToModels(response.data);
                    super.toCsv(filename, data, columns);
                },
                error: () => {
                    this.modalInfoService.openErrorInfoModal(
                        this.transloco.translate('policies.Policies.ExportPolicies.ErrorText.Error')
                    );
                }
            });
    }
}
