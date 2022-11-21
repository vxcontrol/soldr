import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';

import { toHttpParams } from '@soldr/shared';

import { BASE_URL } from '../base-urls';
import { ListQuery, PrivateTags, Response } from '../dto';

@Injectable({
    providedIn: 'root'
})
export class TagsService {
    private readonly baseUrl = `${BASE_URL}/tags/`;

    constructor(private http: HttpClient) {}

    /**
     * Для фильтрации по сущности используется фильтр по полю type (agents, groups, policies, modules)
     *
     * @param query
     */
    fetchList(query: ListQuery) {
        return this.http.get(this.baseUrl, { params: toHttpParams(query) }) as Response<PrivateTags>;
    }
}
