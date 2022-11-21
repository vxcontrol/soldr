import { createAction, props } from '@ngrx/store';

export enum ActionType {
    FetchTags = '[tags] Fetch tags',
    FetchTagsSuccess = '[tags] Fetch tags success',
    FetchTagsFailure = '[tags] Fetch tags failure'
}

export const fetchTags = createAction(ActionType.FetchTags, props<{ domain: string }>());
export const fetchTagsSuccess = createAction(ActionType.FetchTagsSuccess, props<{ domain: string; tags: string[] }>());
export const fetchTagsFailure = createAction(ActionType.FetchTagsFailure, props<{ domain: string }>());
