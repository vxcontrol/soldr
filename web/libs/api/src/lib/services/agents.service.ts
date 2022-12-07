import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';

import { toHttpParams } from '@soldr/shared';

import { BASE_URL } from '../base-urls';
import {
    GroupedListQuery,
    ListQuery,
    ModelsAgent,
    ModelsModuleA,
    PrivateAgent,
    PrivateAgentCountResponse,
    PrivateAgentModules,
    PrivateAgents,
    PrivateAgentsAction,
    PrivateAgentsActionResult,
    PrivatePatchAgentAction,
    Response,
    ResponseList
} from '../dto';

@Injectable({
    providedIn: 'root'
})
export class AgentsService {
    private readonly baseUrl = `${BASE_URL}/agents/`;

    constructor(private http: HttpClient) {}

    fetchList<T extends ListQuery | GroupedListQuery>(query: T): ResponseList<T, PrivateAgents> {
        return this.http.get(this.baseUrl, { params: toHttpParams(query) });
    }

    fetchStatistics() {
        return this.http.get(`${this.baseUrl}count`) as Response<PrivateAgentCountResponse>;
    }

    fetchOne(hash: string) {
        return this.http.get(`${this.baseUrl}${hash}`) as Response<PrivateAgent>;
    }

    doAction(data: PrivateAgentsAction) {
        return this.http.put(this.baseUrl, data) as Response<PrivateAgentsActionResult>;
    }

    delete(hash: string) {
        return this.http.delete(`${this.baseUrl}${hash}`) as Response<any>;
    }

    update(hash: string, data: PrivatePatchAgentAction) {
        return this.http.put(`${this.baseUrl}${hash}`, data) as Response<ModelsAgent>;
    }

    fetchModules(hash: string, query: ListQuery) {
        return this.http.get(`${this.baseUrl}${hash}/modules`, {
            params: toHttpParams(query)
        }) as Response<PrivateAgentModules>;
    }

    fetchModule(hash: string, moduleName: string) {
        return this.http.get(`${this.baseUrl}${hash}/modules/${moduleName}`) as Response<ModelsModuleA>;
    }
}
