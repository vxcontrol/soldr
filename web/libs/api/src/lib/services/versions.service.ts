import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';

import { allListQuery } from '@soldr/api';
import { toHttpParams } from '@soldr/shared';

import { BASE_URL } from '../base-urls';
import { PrivateVersions, SuccessResponse } from '../dto';

@Injectable({
    providedIn: 'root'
})
export class VersionsService {
    constructor(private http: HttpClient) {}

    getVersions(domain: 'agents'): Observable<SuccessResponse<PrivateVersions>> {
        const query = allListQuery({ filters: [{ field: 'type', value: domain }] });

        return this.http.get(`${BASE_URL}/versions/`, { params: toHttpParams(query) }) as Observable<
            SuccessResponse<PrivateVersions>
        >;
    }
}
