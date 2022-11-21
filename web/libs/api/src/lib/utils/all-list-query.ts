import { GroupedListQuery, injectorInstance, ListQuery, ListQueryType } from '@soldr/api';
import { LanguageService } from '@soldr/shared';

export function allListQuery(
    initial: Partial<ListQuery> = {},
    languageService = injectorInstance.get<LanguageService>(LanguageService)
): ListQuery {
    return {
        lang: languageService.lang,
        type: ListQueryType.Init,
        page: 1,
        pageSize: -1,
        sort: {},
        ...initial
    } as ListQuery;
}

export function allGroupedListQuery(initial: Partial<ListQuery>, group: string): GroupedListQuery {
    return {
        ...allListQuery(initial),
        group
    };
}
