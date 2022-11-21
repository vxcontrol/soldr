import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';

import { toHttpParams } from '@soldr/shared';

import { BASE_URL } from '../base-urls';
import {
    GroupedListQuery,
    ListQuery,
    ModelsGroup,
    ModelsModuleA,
    PrivateGroup,
    PrivateGroupInfo,
    PrivateGroupModules,
    PrivateGroupPolicyPatch,
    PrivateGroups,
    Response,
    ResponseList
} from '../dto';

@Injectable({
    providedIn: 'root'
})
export class GroupsService {
    private readonly baseUrl = `${BASE_URL}/groups/`;

    constructor(private http: HttpClient) {}

    fetchList<T extends ListQuery | GroupedListQuery>(query: T): ResponseList<T, PrivateGroups> {
        return this.http.get(this.baseUrl, { params: toHttpParams(query) });
    }

    fetchOne(hash: string) {
        return this.http.get(`${this.baseUrl}${hash}`) as Response<PrivateGroup>;
    }

    create(data: PrivateGroupInfo) {
        return this.http.post(this.baseUrl, data) as Response<ModelsGroup>;
    }

    delete(hash: string) {
        return this.http.delete(`${this.baseUrl}${hash}`) as Response<any>;
    }

    update(hash: string, data: ModelsGroup) {
        return this.http.put(`${this.baseUrl}${hash}`, data) as Response<ModelsGroup>;
    }

    fetchModules(hash: string, query: ListQuery) {
        return this.http.get(`${this.baseUrl}${hash}/modules`, {
            params: toHttpParams(query)
        }) as Response<PrivateGroupModules>;
    }

    fetchModule(hash: string, moduleName: string) {
        return this.http.get(`${this.baseUrl}${hash}/modules/${moduleName}`) as Response<ModelsModuleA>;
    }

    updatePolicy(hash: string, data: PrivateGroupPolicyPatch) {
        return this.http.put(`${this.baseUrl}${hash}/policies`, data) as Response<any>;
    }
}
