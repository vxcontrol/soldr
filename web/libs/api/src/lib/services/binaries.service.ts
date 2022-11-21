import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { map, Observable } from 'rxjs';

import { Architecture, OperationSystem, toHttpParams } from '@soldr/shared';

import { BASE_URL } from '../base-urls';
import { ListQuery, PrivateBinaries, SuccessResponse } from '../dto';

@Injectable({
    providedIn: 'root'
})
export class BinariesService {
    constructor(private http: HttpClient) {}

    getBinaries(query: ListQuery): Observable<SuccessResponse<PrivateBinaries>> {
        return this.http.get(`${BASE_URL}/binaries/vxagent`, { params: toHttpParams(query) }) as Observable<
            SuccessResponse<PrivateBinaries>
        >;
    }

    getBinaryFile(os: OperationSystem, arch: Architecture, version = 'latest') {
        return this.http
            .get(`${BASE_URL}/binaries/vxagent/${os}/${arch}/${version}`, {
                observe: 'response',
                responseType: 'arraybuffer'
            })
            .pipe(
                map((response) => {
                    const blob = new Blob([response.body], { type: 'octet/stream' });

                    return {
                        name: response.headers.get('content-disposition').split('filename=')[1].split('"')[1],
                        url: window.URL.createObjectURL(blob)
                    };
                })
            );
    }
}
