import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';

import { toHttpParams } from '@soldr/shared';

import { BASE_URL } from '../base-urls';
import {
    GroupedListQuery,
    ListQuery,
    ModelsGroup,
    ModelsModuleA,
    ModelsModuleConfig,
    ModelsPolicy,
    PrivatePolicies,
    PrivatePolicy,
    PrivatePolicyCountResponse,
    PrivatePolicyGroupPatch,
    PrivatePolicyInfo,
    PrivatePolicyModules,
    Response,
    ResponseList
} from '../dto';

@Injectable({
    providedIn: 'root'
})
export class PoliciesService {
    private readonly baseUrl = `${BASE_URL}/policies/`;

    constructor(private http: HttpClient) {}

    fetchList<T extends ListQuery | GroupedListQuery>(query: T): ResponseList<T, PrivatePolicies> {
        return this.http.get(this.baseUrl, { params: toHttpParams(query) });
    }

    fetchStatistics() {
        return this.http.get(`${this.baseUrl}count`) as Response<PrivatePolicyCountResponse>;
    }

    fetchOne(hash: string) {
        return this.http.get(`${this.baseUrl}${hash}`) as Response<PrivatePolicy>;
    }

    create(data: PrivatePolicyInfo) {
        return this.http.post(this.baseUrl, data) as Response<ModelsPolicy>;
    }

    delete(hash: string) {
        return this.http.delete(`${this.baseUrl}${hash}`) as Response<any>;
    }

    update(hash: string, data: ModelsGroup) {
        return this.http.put(`${this.baseUrl}${hash}`, data) as Response<ModelsPolicy>;
    }

    fetchModules(hash: string, query: ListQuery) {
        return this.http.get(`${this.baseUrl}${hash}/modules`, {
            params: toHttpParams(query)
        }) as Response<PrivatePolicyModules>;
    }

    fetchModule(hash: string, moduleName: string) {
        return this.http.get(`${this.baseUrl}${hash}/modules/${moduleName}`) as Response<ModelsModuleA>;
    }

    updateGroup(hash: string, data: PrivatePolicyGroupPatch) {
        return this.http.put(`${this.baseUrl}${hash}/groups`, data) as Response<any>;
    }

    activateModule(hash: string, moduleName: string) {
        const params = { action: 'activate' };

        return this.http.put(`${this.baseUrl}${hash}/modules/${moduleName}`, params) as Response<any>;
    }

    deactivateModule(hash: string, moduleName: string) {
        const params = { action: 'deactivate' };

        return this.http.put(`${this.baseUrl}${hash}/modules/${moduleName}`, params) as Response<any>;
    }

    updateModule(hash: string, moduleName: string, version?: string) {
        const params: { action: string; version?: string } = { action: 'update' };

        if (version) {
            params.version = version;
        }

        return this.http.put(`${this.baseUrl}${hash}/modules/${moduleName}`, params) as Response<any>;
    }

    storeModule(hash: string, module: ModelsModuleA) {
        const params = { action: 'store', module };

        return this.http.put(`${this.baseUrl}${hash}/modules/${module.info.name}`, params) as Response<any>;
    }

    changeModuleVersion(hash: string, moduleName: string, version: string) {
        const params = { action: 'update', version };

        return this.http.put(`${this.baseUrl}${hash}/modules/${moduleName}`, params) as Response<any>;
    }

    deleteModule(hash: string, moduleName: string) {
        return this.http.delete(`${this.baseUrl}${hash}/modules/${moduleName}`) as Response<any>;
    }

    updateSecureParams(hash: string, moduleName: string, data: any) {
        return this.http.post(`${this.baseUrl}${hash}/modules/${moduleName}/secure_config`, data) as Response<any>;
    }

    getSecureParam(hash: string, moduleName: string, paramName: string) {
        return this.http.get(
            `${this.baseUrl}${hash}/modules/${moduleName}/secure_config/${paramName}`
        ) as Response<ModelsModuleConfig>;
    }
}
