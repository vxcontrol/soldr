import { Injectable } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';
import { shareReplay, switchMap, take } from 'rxjs';

import { AgentsService, PrivateAgents, SuccessResponse } from '@soldr/api';
import { Agent, manyAgentsToModels } from '@soldr/models';
import {
    AgentVersionPipe,
    ExporterService,
    LanguageService,
    ModalInfoService,
    OperationSystem,
    OsFormatterService
} from '@soldr/shared';
import { AgentListFacade } from '@soldr/store/agents';

@Injectable({
    providedIn: 'root'
})
export class AgentsExporterService extends ExporterService {
    // eslint-disable @typescript-eslint/naming-convention
    mapper: Record<string, (value: any) => string> = {
        status: (agent: Agent) => {
            if (agent.status === 'connected') {
                return this.transloco.translate('shared.Shared.AgentConnectionStatus.TooltipText.Connected');
            } else if (agent.status === 'disconnected') {
                return this.transloco.translate('shared.Shared.AgentConnectionStatus.TooltipText.Disconnected');
            }

            return '';
        },

        os: (agent: Agent) => {
            const family = Object.keys(agent.info.os)[0] as OperationSystem;

            return this.osFormatter.getText(family, agent.info.os[family]);
        },

        auth_status: (agent: Agent) => {
            switch (agent.auth_status) {
                case 'authorized':
                    return this.transloco.translate('shared.Shared.AgentAuthStatus.TooltipText.Authorized');
                case 'unauthorized':
                    return this.transloco.translate('shared.Shared.AgentAuthStatus.TooltipText.Unauthorized');
                case 'blocked':
                    return this.transloco.translate('shared.Shared.AgentAuthStatus.TooltipText.Blocked');
                default:
                    return '';
            }
        },

        'details.upgrade_task': (agent: Agent) => {
            switch (agent.details?.upgrade_task?.status) {
                case 'new':
                case 'running':
                    return this.transloco.translate('agents.Agents.Pseudo.Text.UpgradeInProcess', {
                        version: agent.details?.upgrade_task?.version
                    });
                case 'ready':
                    return this.transloco.translate('agents.Agents.Pseudo.Text.UpgradeIsSuccessful', {
                        version: agent.details?.upgrade_task?.version
                    });
                case 'failed':
                    return this.transloco.translate('agents.Agents.Pseudo.Text.UpgradeIsFailed');
                default:
                    return new AgentVersionPipe().transform(agent.version);
            }
        },

        'details.modules': (agent: Agent) =>
            agent.details.modules?.map((module) => module.locale.module[this.languageService.lang].title).join(', '),

        'details.group': (agent: Agent) => agent.details.group?.info.name[this.languageService.lang],

        'info.tags': (agent: Agent) => agent.info.tags.join(', ')
    };
    headers: Record<string, () => string> = {
        description: () => this.transloco.translate('agents.Agents.AgentsList.GridColumnsTitle.Name'),
        ip: () => this.transloco.translate('agents.Agents.AgentsList.GridColumnsTitle.Ip'),
        status: () => this.transloco.translate('agents.Agents.AgentsList.GridColumnsTitle.Connection'),
        os: () => this.transloco.translate('agents.Agents.AgentsList.GridColumnsTitle.Os'),
        auth_status: () => this.transloco.translate('agents.Agents.AgentsList.GridColumnsTitle.Authorization'),
        'details.upgrade_task': () => this.transloco.translate('agents.Agents.AgentsList.GridColumnsTitle.Version'),
        'details.modules': () => this.transloco.translate('agents.Agents.AgentsList.GridColumnsTitle.Modules'),
        'details.group': () => this.transloco.translate('agents.Agents.AgentsList.GridColumnsTitle.Group'),
        'info.tags': () => this.transloco.translate('shared.Shared.Pseudo.GridColumnsTitle.Tags'),
        connected_date: () => this.transloco.translate('agents.Agents.AgentsList.GridColumnsTitle.LastConnection'),
        hash: () => this.transloco.translate('agents.Agents.AgentsList.GridColumnsTitle.Id')
    };

    // eslint-enable @typescript-eslint/naming-convention

    constructor(
        private agentListFacade: AgentListFacade,
        private agentsService: AgentsService,
        private exporterService: ExporterService,
        private modalInfoService: ModalInfoService,
        private transloco: TranslocoService,
        private osFormatter: OsFormatterService,
        private languageService: LanguageService
    ) {
        super();
    }

    export(columns: string[], agents?: Agent[]) {
        const filename = this.transloco.translate('agents.Agents.ExportAgents.Text.ExportedFileName');
        this.agentListFacade.allListQuery$
            .pipe(
                shareReplay({ bufferSize: 1, refCount: true }),
                take(1),
                switchMap((initialQuery) => {
                    const query =
                        agents?.length > 0
                            ? {
                                  ...initialQuery,
                                  filters: [...initialQuery.filters, { field: 'id', value: agents.map(({ id }) => id) }]
                              }
                            : initialQuery;

                    return this.agentsService.fetchList(query);
                })
            )
            .subscribe({
                next: (response: SuccessResponse<PrivateAgents>) => {
                    const data = manyAgentsToModels(response.data);
                    super.toCsv(filename, data, columns);
                },
                error: () => {
                    this.modalInfoService.openErrorInfoModal(
                        this.transloco.translate('agents.Agents.ExportAgents.ErrorText.Error')
                    );
                }
            });
    }
}
