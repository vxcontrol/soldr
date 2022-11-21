import { createReducer, on } from '@ngrx/store';

import * as TagsActions from './tags.actions';

export const tagsFeatureKey = 'tags';

export interface State {
    tags: Record<string, string[]>;
    isLoadingTags: Record<string, boolean>;
}

export const initialState: State = { isLoadingTags: {}, tags: {} };

export const reducer = createReducer(
    initialState,

    on(TagsActions.fetchTags, (state, { domain }) => ({
        ...state,
        isLoadingTags: {
            ...state.isLoadingTags,
            [domain]: true
        }
    })),
    on(TagsActions.fetchTagsSuccess, (state, { domain, tags }) => ({
        ...state,
        isLoadingTags: {
            ...state.isLoadingTags,
            [domain]: false
        },
        tags: {
            ...state.tags,
            [domain]: tags
        }
    })),
    on(TagsActions.fetchTagsFailure, (state, { domain }) => ({
        ...state,
        isLoadingTags: {
            ...state.isLoadingTags,
            [domain]: false
        }
    }))
);
