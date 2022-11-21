import { createFeatureSelector, createSelector } from '@ngrx/store';

import * as fromTags from './tags.reducer';

export const selectTagsState = createFeatureSelector<fromTags.State>(fromTags.tagsFeatureKey);
export const selectAgentsTags = createSelector(selectTagsState, (state) => state.tags.agents || []);
export const selectGroupsTags = createSelector(selectTagsState, (state) => state.tags.groups || []);
export const selectPoliciesTags = createSelector(selectTagsState, (state) => state.tags.policies || []);
export const selectModulesTags = createSelector(selectTagsState, (state) => state.tags.modules || []);
export const selectIsLoadingAgentsTags = createSelector(selectTagsState, (state) => state.isLoadingTags.agents);
export const selectIsLoadingGroupsTags = createSelector(selectTagsState, (state) => state.isLoadingTags.groups);
export const selectIsLoadingPoliciesTags = createSelector(selectTagsState, (state) => state.isLoadingTags.policies);
export const selectIsLoadingModulesTags = createSelector(selectTagsState, (state) => state.isLoadingTags.modules);
