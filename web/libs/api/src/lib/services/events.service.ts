import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';

import { toHttpParams } from '@soldr/shared';

import { BASE_URL } from '../base-urls';
import { GroupedListQuery, ListQuery, PrivateEvents, ResponseList } from '../dto';

@Injectable({
    providedIn: 'root'
})
export class EventsService {
    private readonly baseUrl = `${BASE_URL}/events/`;

    constructor(private http: HttpClient) {}

    fetchEvents<T extends ListQuery | GroupedListQuery>(query: T): ResponseList<T, PrivateEvents> {
        return this.http.get(this.baseUrl, { params: toHttpParams(query) });
    }
}
