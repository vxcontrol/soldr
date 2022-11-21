import { HttpParams } from '@angular/common/http';

import { ListQuery } from '@soldr/api';

export function toHttpParams(query: ListQuery): HttpParams {
    let params = new HttpParams()
        .set('page', query.page)
        .set('pageSize', query.pageSize)
        .set('type', query.type)
        .set('sort', JSON.stringify(query.sort))
        .set('lang', query.lang);

    if (query.group) {
        params = params.set('group', query.group);
    }

    for (const filter of query.filters || []) {
        params = params.append('filters[]', JSON.stringify(filter));
    }

    return params;
}
