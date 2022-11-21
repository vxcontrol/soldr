import { Observable } from 'rxjs';

import { Filtration } from '@soldr/shared';

export enum StatusResponse {
    Success = 'success',
    Error = 'error'
}

export interface ErrorResponse {
    code: string;
    error: string;
    /** @example error message text */
    msg?: string;

    /** @example error */
    status?: StatusResponse;
}

export interface SuccessResponse<T> {
    data?: T;

    /** @example success */
    status?: StatusResponse;
}

export type Response<T> = Observable<SuccessResponse<T> | ErrorResponse>;

export type ResponseList<T, D> = T extends GroupedListQuery ? Response<GroupedData> : Response<D>;

export enum ListQueryType {
    Sort = 'sort',
    Filter = 'filter',
    Init = 'init',
    Page = 'page',
    Size = 'size'
}

export enum ListQueryLang {
    En = 'en',
    Ru = 'ru'
}

export interface GroupedData {
    grouped: string[];
    total: number;
}

export interface ListQuery {
    filters?: Filtration[];
    group?: string;
    lang: ListQueryLang;
    page: number;
    pageSize: number;
    sort: { prop: string; order: string } | Record<string, never>;
    type: ListQueryType;
}

export interface GroupedListQuery extends ListQuery {
    group: string;
}
