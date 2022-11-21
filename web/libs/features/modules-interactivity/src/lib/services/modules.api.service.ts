// @ts-ignore
import * as httpVueLoader from 'http-vue-loader';

import {
    AgentsService,
    allListQuery,
    EventsService,
    GroupsService,
    ListQuery,
    ModelsModuleA,
    PoliciesService,
    Response
} from '@soldr/api';
import { ViewMode } from '@soldr/shared';

import { EntityModules } from '../types/entity-modules';

const UNSUPPORTED_OPERATION_ERROR = 'Unsupported operation for current view mode';

export class ModulesApiService {
    constructor(
        private props: { id: number; hash: string; module_name: string; viewMode: ViewMode },
        private agentsService: AgentsService,
        private groupsService: GroupsService,
        private policiesService: PoliciesService
    ) {}

    getModules(params: ListQuery) {
        return new Promise((resolve, reject) => {
            params.sort = params.sort && params.sort.prop ? params.sort : { prop: 'name', order: 'ascending' };

            const response$: Response<EntityModules> = this.getService(this.props.viewMode).fetchModules(
                this.props.hash,
                allListQuery(params)
            );

            response$.subscribe({
                next: (response) => resolve(response),
                error: (response: unknown) => reject(response)
            });
        });
    }

    getModule(name: string) {
        return new Promise((resolve, reject) => {
            const response$: Response<ModelsModuleA> = this.getService(this.props.viewMode).fetchModule(
                this.props.hash,
                name
            );

            response$.subscribe({
                next: (response) => resolve(response),
                error: (response: unknown) => reject(response)
            });
        });
    }

    getView(name: string, fileName?: string) {
        const query = fileName ? `?file=${fileName}` : '';

        return httpVueLoader(`/api/v1/${this.props.viewMode}/${this.props.hash}/modules/${name}/bmodule.vue${query}`);
    }

    activateModule(name: string) {
        if (this.props.viewMode === ViewMode.Policies) {
            return new Promise((resolve, reject) => {
                this.policiesService.activateModule(this.props.hash, name).subscribe({
                    next: (response) => resolve(response),
                    error: (response: unknown) => reject(response)
                });
            });
        }

        throw new Error(UNSUPPORTED_OPERATION_ERROR);
    }

    deactivateModule(name: string) {
        if (this.props.viewMode === ViewMode.Policies) {
            return new Promise((resolve, reject) => {
                this.policiesService.deactivateModule(this.props.hash, name).subscribe({
                    next: (response) => resolve(response),
                    error: (response: unknown) => reject(response)
                });
            });
        }

        throw new Error(UNSUPPORTED_OPERATION_ERROR);
    }

    updateModule(name: string) {
        if (this.props.viewMode === ViewMode.Policies) {
            return new Promise((resolve, reject) => {
                this.policiesService.updateModule(this.props.hash, name).subscribe({
                    next: (response) => resolve(response),
                    error: (response: unknown) => reject(response)
                });
            });
        }

        throw new Error(UNSUPPORTED_OPERATION_ERROR);
    }

    changeModuleVersion(name: string, version: string) {
        if (this.props.viewMode === ViewMode.Policies) {
            return new Promise((resolve, reject) => {
                this.policiesService.updateModule(this.props.hash, name, version).subscribe({
                    next: (response) => resolve(response),
                    error: (response: unknown) => reject(response)
                });
            });
        }

        throw new Error(UNSUPPORTED_OPERATION_ERROR);
    }

    deleteModule(name: string) {
        if (this.props.viewMode === ViewMode.Policies) {
            return new Promise((resolve, reject) => {
                this.policiesService.deleteModule(this.props.hash, name).subscribe({
                    next: (response) => resolve(response),
                    error: (response: unknown) => reject(response)
                });
            });
        }

        throw new Error(UNSUPPORTED_OPERATION_ERROR);
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
}
