import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';

import { toHttpParams } from '@soldr/shared';

import { BASE_URL } from '../base-urls';
import {
    ListQuery,
    ModelsAgentUpgradeTask,
    PrivateUpgradeAgent,
    PrivateUpgradesAgents,
    PrivateUpgradesAgentsAction,
    PrivateUpgradesAgentsActionResult,
    Response
} from '../dto';

@Injectable({
    providedIn: 'root'
})
export class UpgradesService {
    private readonly baseUrl = `${BASE_URL}/upgrades`;

    constructor(private http: HttpClient) {}

    fetchList(query: ListQuery) {
        return this.http.get(`${this.baseUrl}/agents`, {
            params: toHttpParams(query)
        }) as Response<PrivateUpgradesAgents>;
    }

    upgradeAgent(data: PrivateUpgradesAgentsAction) {
        return this.http.post(`${this.baseUrl}/agents`, {
            ...data,
            version: data.version || 'latest'
        }) as Response<PrivateUpgradesAgentsActionResult>;
    }

    fetchLastAgentDetails(hash: string) {
        return this.http.get(`${this.baseUrl}/agents/${hash}/last`) as Response<PrivateUpgradeAgent>;
    }

    updateLastAgentDetails(hash: string, data: ModelsAgentUpgradeTask) {
        return this.http.put(`${this.baseUrl}/agents/${hash}/last`, {
            ...data,
            version: data.version || 'latest'
        }) as Response<ModelsAgentUpgradeTask>;
    }
}
