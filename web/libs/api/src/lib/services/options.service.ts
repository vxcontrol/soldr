import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';

import {
    ListQuery,
    PrivateOptionsActions,
    PrivateOptionsEvents,
    PrivateOptionsFields,
    PrivateOptionsTags,
    Response
} from '@soldr/api';
import { toHttpParams } from '@soldr/shared';

import { BASE_URL } from '../base-urls';

@Injectable({
    providedIn: 'root'
})
export class OptionsService {
    private readonly baseUrl = `${BASE_URL}/options/`;

    constructor(private http: HttpClient) {}

    fetchOptionsActions(query: ListQuery) {
        return this.http.get(`${this.baseUrl}actions`, {
            params: toHttpParams(query)
        }) as Response<PrivateOptionsActions>;
    }

    fetchOptionsEvents(query: ListQuery) {
        return this.http.get(`${this.baseUrl}events`, {
            params: toHttpParams(query)
        }) as Response<PrivateOptionsEvents>;
    }

    fetchOptionsFields(query: ListQuery) {
        return this.http.get(`${this.baseUrl}fields`, {
            params: toHttpParams(query)
        }) as Response<PrivateOptionsFields>;
    }

    fetchOptionsTags(query: ListQuery) {
        return this.http.get(`${this.baseUrl}tags`, { params: toHttpParams(query) }) as Response<PrivateOptionsTags>;
    }
}
