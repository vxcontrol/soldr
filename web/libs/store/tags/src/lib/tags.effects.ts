import { Injectable } from '@angular/core';
import { Actions, createEffect, ofType } from '@ngrx/effects';
import { map, mergeMap } from 'rxjs';

import { allListQuery, PrivateTags, SuccessResponse, TagsService } from '@soldr/api';

import * as TagsActions from './tags.actions';

@Injectable()
export class TagsEffects {
    fetchTags$ = createEffect(() =>
        this.actions$.pipe(
            ofType(TagsActions.fetchTags),
            mergeMap(({ domain }) =>
                this.tagsService
                    .fetchList(allListQuery({ filters: [{ field: 'type', value: domain }] }))
                    .pipe(
                        map((response: SuccessResponse<PrivateTags>) =>
                            TagsActions.fetchTagsSuccess({ domain, tags: response.data?.tags })
                        )
                    )
            )
        )
    );

    constructor(private actions$: Actions, private tagsService: TagsService) {}
}
