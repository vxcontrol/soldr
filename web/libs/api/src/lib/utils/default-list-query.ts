import { injectorInstance, ListQuery, ListQueryType } from '@soldr/api';
import { LanguageService, PAGE_SIZE } from '@soldr/shared';

export function defaultListQuery(
    initial: Partial<ListQuery> = {},
    languageService = injectorInstance.get<LanguageService>(LanguageService)
): ListQuery {
    return {
        lang: languageService.lang,
        type: ListQueryType.Init,
        page: 1,
        pageSize: PAGE_SIZE,
        sort: {},
        ...initial
    } as ListQuery;
}
