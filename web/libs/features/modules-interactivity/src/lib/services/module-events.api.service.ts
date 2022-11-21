import {
    AgentsService,
    allListQuery,
    EventsService,
    GroupsService,
    ListQuery,
    PoliciesService,
    PrivateEvents,
    Response,
    SuccessResponse
} from '@soldr/api';
import { ViewMode } from '@soldr/shared';

import { EntityModules } from '../types/entity-modules';

export class ModuleEventsApiService {
    constructor(
        private props: { id: number; hash: string; module_name: string; viewMode: ViewMode },
        private agentsService: AgentsService,
        private eventsService: EventsService,
        private groupsService: GroupsService,
        private policiesService: PoliciesService
    ) {}

    getEvents(params: ListQuery) {
        params.sort = params.sort && params.sort.prop ? params.sort : { prop: 'localizedDate', order: 'descending' };
        params.filters = params.filters || [];
        params.filters.push({ field: this.getEntityFieldId(this.props.viewMode), value: this.props.id });

        if (this.props.module_name) {
            params.filters.push({ field: 'module_name', value: this.props.module_name });
        }

        return new Promise((resolve, reject) => {
            this.eventsService.fetchEvents(params).subscribe({
                next: (response: SuccessResponse<PrivateEvents>) => resolve(response),
                error: (response: unknown) => reject(response)
            });
        });
    }

    getModules() {
        return new Promise((resolve, reject) => {
            const response$: Response<EntityModules> = this.getService(this.props.viewMode).fetchModules(
                this.props.hash,
                allListQuery()
            );

            response$.subscribe({
                next: (response) => resolve(response),
                error: (response: unknown) => reject(response)
            });
        });
    }

    getGroups() {
        if (this.props.viewMode === ViewMode.Policies) {
            return new Promise((resolve, reject) => {
                this.groupsService
                    .fetchList(
                        allListQuery({
                            filters: [{ field: 'policy_id', value: this.props.id }]
                        })
                    )
                    .subscribe({
                        next: (response) => resolve(response),
                        error: (response: unknown) => reject(response)
                    });
            });
        }

        return Promise.resolve([]);
    }

    getPolicies() {
        if (this.props.viewMode === ViewMode.Groups) {
            return new Promise((resolve, reject) => {
                this.policiesService
                    .fetchList(
                        allListQuery({
                            filters: [{ field: 'group_id', value: this.props.id }]
                        })
                    )
                    .subscribe({
                        next: (response) => resolve(response),
                        error: (response: unknown) => reject(response)
                    });
            });
        }

        return Promise.resolve([]);
    }

    getAgents() {
        if ([ViewMode.Policies, ViewMode.Groups].includes(this.props.viewMode)) {
            return new Promise((resolve, reject) => {
                const filterField = this.props.viewMode === ViewMode.Policies ? 'policy_id' : 'group_id';
                const filter = { field: filterField, value: this.props.id };

                this.agentsService
                    .fetchList(
                        allListQuery({
                            filters: [filter]
                        })
                    )
                    .subscribe({
                        next: (response) => resolve(response),
                        error: (response: unknown) => reject(response)
                    });
            });
        }

        return Promise.resolve([]);
    }

    private getService(viewMode: ViewMode): AgentsService | PoliciesService | GroupsService {
        switch (viewMode) {
            case ViewMode.Agents:
                return this.agentsService;
            case ViewMode.Groups:
                return this.groupsService;
            case ViewMode.Policies:
                return this.policiesService;
        }
    }

    private getEntityFieldId(viewMode: ViewMode): string {
        switch (viewMode) {
            case ViewMode.Agents:
                return 'agent_id';
            case ViewMode.Groups:
                return 'group_id';
            case ViewMode.Policies:
                return 'policy_id';
        }
    }
}
