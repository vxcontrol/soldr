import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';

import { UploadModule } from '@soldr/features/modules';
import { toHttpParams } from '@soldr/shared';

import { BASE_URL } from '../base-urls';
import {
    ListQuery,
    ModelsChangelogVersion,
    ModelsModuleInfo,
    ModelsModuleS,
    PrivateModuleVersionPatch,
    PrivatePolicyModulesUpdates,
    PrivateSystemModuleFile,
    PrivateSystemModuleFilePatch,
    PrivateSystemModules,
    PrivateSystemShortModules,
    Response
} from '../dto';
import { allListQuery } from '../utils';

@Injectable({
    providedIn: 'root'
})
export class ModulesService {
    private readonly baseUrl = `${BASE_URL}/modules/`;
    private readonly importBaseUrl = `${BASE_URL}/import/modules/`;
    private readonly exportBaseUrl = `${BASE_URL}/export/modules/`;

    constructor(private http: HttpClient) {}

    importModule(name: string, version = 'all', data: UploadModule) {
        const options = {
            params: { rewrite: data.rewrite }
        };

        return this.http.post(`${this.importBaseUrl}${name}/versions/${version}`, data.archive, options);
    }

    exportModule(name: string, version = 'all') {
        return this.http.post(`${this.exportBaseUrl}${name}/versions/${version}`, null, {
            observe: 'response',
            responseType: 'arraybuffer'
        });
    }

    createDraft(name: string, version: string, changelog: ModelsChangelogVersion) {
        return this.http.post(`${this.baseUrl}${name}/versions/${version}`, changelog);
    }

    fetchList(query: ListQuery) {
        return this.http.get(this.baseUrl, { params: toHttpParams(query) }) as Response<PrivateSystemModules>;
    }

    fetchOne(name: string, version: string) {
        return this.http.get(`${this.baseUrl}${name}/versions/${version}`) as Response<ModelsModuleS>;
    }

    fetchUpdates(name: string, version: string) {
        return this.http.get(
            `${this.baseUrl}${name}/versions/${version}/updates`
        ) as Response<PrivatePolicyModulesUpdates>;
    }

    create(data: ModelsModuleInfo) {
        return this.http.post(this.baseUrl, data);
    }

    update(name: string, version: string, data: PrivateModuleVersionPatch) {
        return this.http.put(`${this.baseUrl}${name}/versions/${version}`, data) as Response<any>;
    }

    deleteModule(name: string) {
        return this.http.delete(`${this.baseUrl}${name}`) as Response<any>;
    }

    deleteModuleVersion(name: string, version: string) {
        return this.http.delete(`${this.baseUrl}${name}/versions/${version}`) as Response<any>;
    }

    fetchVersions(name: string, query = allListQuery()) {
        return this.http.get(`${this.baseUrl}${name}/versions`, {
            params: toHttpParams(query)
        }) as Response<PrivateSystemShortModules>;
    }

    updateInPolitics(name: string, version: string = 'latest') {
        return this.http.post(`${this.baseUrl}${name}/versions/${version}/updates`, {});
    }

    fetchFiles(name: string, version: string) {
        return this.http.get(`${this.baseUrl}${name}/versions/${version}/files`) as Response<string[]>;
    }

    fetchFile(name: string, version: string, filename: string) {
        return this.http.get(`${this.baseUrl}${name}/versions/${version}/files/file`, {
            params: {
                path: filename
            }
        }) as Response<PrivateSystemModuleFile>;
    }

    patchFile(name: string, version: string, patch: PrivateSystemModuleFilePatch) {
        return this.http.put(
            `${this.baseUrl}${name}/versions/${version}/files/file`,
            patch
        ) as Response<PrivateSystemModuleFile>;
    }
}
