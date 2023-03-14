import { createFeatureSelector, createSelector } from '@ngrx/store';

import { ServiceStatus } from '@soldr/api';

import * as fromShared from './shared.reducer';
import { State } from './shared.reducer';

export const selectSharedState = createFeatureSelector<fromShared.State>(fromShared.sharedFeatureKey);

export const selectInfo = createSelector(selectSharedState, (state: State) => state.info);
export const selectIsInfoLoading = createSelector(selectSharedState, (state: State) => state.isInfoLoading);
export const selectIsInfoLoaded = createSelector(selectSharedState, (state: State) => state.isInfoLoaded);
export const selectIsAuthorized = createSelector(selectSharedState, (state: State) => state.info?.type === 'user');
export const selectLatestBinary = createSelector(selectSharedState, (state: State) => state.latestAgentBinary);
export const selectIsLoadingLatestBinary = createSelector(
    selectSharedState,
    (state: State) => state.isLoadingLatestAgentBinary
);
export const selectBinaries = createSelector(selectSharedState, (state: State) => state.agentBinaries);
export const selectIsLoadingBinaries = createSelector(
    selectSharedState,
    (state: State) => state.isLoadingAgentBinaries
);

export const selectIsExportingBinaryFile = createSelector(
    selectSharedState,
    (state: State) => state.isExportingBinaryFile
);
export const selectExportError = createSelector(selectSharedState, (state: State) => state.exportError);
export const selectActions = createSelector(selectSharedState, (state: State) => state.actions);
export const selectEvents = createSelector(selectSharedState, (state: State) => state.events);
export const selectFields = createSelector(selectSharedState, (state: State) => state.fields);
export const selectTags = createSelector(selectSharedState, (state: State) => state.tags);
export const selectIsLoadingActions = createSelector(
    selectSharedState,
    (state: State) => state.isLoadingOptionsActions
);
export const selectIsLoadingEvents = createSelector(selectSharedState, (state: State) => state.isLoadingOptionsEvents);
export const selectIsLoadingFields = createSelector(selectSharedState, (state: State) => state.isLoadingOptionsFields);
export const selectIsLoadingTags = createSelector(selectSharedState, (state: State) => state.isLoadingOptionsTags);

export const selectAllAgents = createSelector(selectSharedState, (state) => state.allAgents);
export const selectIsLoadingAllAgents = createSelector(selectSharedState, (state) => state.isLoadingAllAgents);
export const selectAllTotalAgents = createSelector(selectSharedState, (state) => state.allTotalAgents);

export const selectAllPolicies = createSelector(selectSharedState, (state) => state.allPolicies);
export const selectAllTotalPolicies = createSelector(selectSharedState, (state) => state.allTotalPolicies);
export const selectIsLoadingAllPolicies = createSelector(selectSharedState, (state) => state.isLoadingAllPolicies);

export const selectIsLoadingAllGroups = createSelector(selectSharedState, (state) => state.isLoadingAllGroups);
export const selectInitializedGroups = createSelector(selectSharedState, (state) => state.initializedGroups);
export const selectAllTotalGroups = createSelector(selectSharedState, (state) => state.allTotalGroups);
export const selectAllGroups = createSelector(selectSharedState, (state) => state.allGroups);

export const selectAllServices = createSelector(selectSharedState, (state: State) => state.allServices);
export const selectIsLoadingAllServices = createSelector(
    selectSharedState,
    (state: State) => state.isLoadingAllServices
);

export const selectAllModules = createSelector(selectSharedState, (state) => state.allModules);
export const selectIsLoadingAllModules = createSelector(selectSharedState, (state) => state.isLoadingAllModules);
export const selectAllTotalModules = createSelector(selectSharedState, (state) => state.allTotalModules);

export const selectSelectedTags = createSelector(selectSharedState, (state) => state.selectedTags);
export const selectSearchValue = createSelector(selectSharedState, (state) => state.searchValue);

export const selectShortServices = createSelector(selectInfo, (info) =>
    info?.services
        ? info.services
              .filter((service) => service.status === ServiceStatus.Active)
              .map((service) => ({
                  name: service?.name,
                  path: `/services/${service?.hash}`
              }))
              .sort((a, b) => (a.name > b.name ? 1 : a.name < b.name ? -1 : 0))
        : []
);

export const selectIsPasswordChangeRequired = createSelector(
    selectSharedState,
    (state) => state.info?.user?.password_change_required
);
export const selectIsChangingPassword = createSelector(selectSharedState, (state) => state.isChangingPassword);
export const selectPasswordChangeError = createSelector(selectSharedState, (state) => state.passwordChangeError);
