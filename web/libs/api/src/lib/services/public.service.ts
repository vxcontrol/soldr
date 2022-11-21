import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';

import { BASE_URL } from '../base-urls';
import { ModelsService, ModelsSignIn, PublicInfo, Response, SuccessResponse } from '../dto';

@Injectable({
    providedIn: 'root'
})
export class PublicService {
    constructor(private http: HttpClient) {}

    getUserInfo(refreshCookie?: boolean): Observable<SuccessResponse<PublicInfo>> {
        const params = !refreshCookie ? new HttpParams().set('refresh_cookie', refreshCookie) : undefined;

        return this.http.get(`${BASE_URL}/info`, { params }) as Observable<SuccessResponse<PublicInfo>>;
    }

    login(data: ModelsSignIn): Observable<SuccessResponse<any>> {
        return this.http.post(`${BASE_URL}/auth/login`, data) as Observable<SuccessResponse<any>>;
    }

    logout(): Observable<SuccessResponse<any>> {
        return this.http.get(`${BASE_URL}/auth/logout`) as Observable<SuccessResponse<any>>;
    }

    switchService(hash: string): Response<ModelsService> {
        const data = new FormData();
        data.append('service', hash);

        return this.http.post(`${BASE_URL}/auth/switch-service`, data);
    }
}
