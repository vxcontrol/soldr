import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';

import { ListQuery, ModelsService, PrivateServices, Response } from '@soldr/api';
import { toHttpParams } from '@soldr/shared';

import { BASE_URL } from '../base-urls';

@Injectable({
    providedIn: 'root'
})
export class ServicesService {
    private readonly baseUrl = `${BASE_URL}/services/`;

    constructor(private http: HttpClient) {}

    fetchServiceList(query: ListQuery) {
        return this.http.get(this.baseUrl, { params: toHttpParams(query) }) as Response<PrivateServices>;
    }

    fetchService(hash: string) {
        return this.http.get(`${this.baseUrl}${hash}`) as Response<ModelsService>;
    }

    createService(data: ModelsService) {
        return this.http.post(this.baseUrl, data) as Response<ModelsService>;
    }

    updateService(data: ModelsService, hash: string) {
        return this.http.put(`${this.baseUrl}${hash}`, data) as Response<ModelsService>;
    }

    deleteService(hash: string) {
        return this.http.delete(`${this.baseUrl}${hash}`) as Response<any>;
    }
}
