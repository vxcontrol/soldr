import { Injectable } from '@angular/core';
import { Store } from '@ngrx/store';

import * as TagsActions from './tags.actions';
import { State } from './tags.reducer';
import {
    selectAgentsTags,
    selectGroupsTags,
    selectIsLoadingAgentsTags,
    selectIsLoadingGroupsTags,
    selectIsLoadingModulesTags,
    selectIsLoadingPoliciesTags,
    selectModulesTags,
    selectPoliciesTags
} from './tags.selectors';

export enum TagDomain {
    Agents = 'agents',
    Groups = 'groups',
    Modules = 'modules',
    Policies = 'policies'
}

@Injectable({
    providedIn: 'root'
})
export class TagsFacade {
    agentsTags$ = this.store.select(selectAgentsTags);
    groupsTags$ = this.store.select(selectGroupsTags);
    policiesTags$ = this.store.select(selectPoliciesTags);
    modulesTags$ = this.store.select(selectModulesTags);
    isLoadingAgentsTags$ = this.store.select(selectIsLoadingAgentsTags);
    isLoadingGroupsTags$ = this.store.select(selectIsLoadingGroupsTags);
    isLoadingPoliciesTags$ = this.store.select(selectIsLoadingPoliciesTags);
    isLoadingModulesTags$ = this.store.select(selectIsLoadingModulesTags);

    constructor(private store: Store<State>) {}

    fetchTags(domain: TagDomain): void {
        this.store.dispatch(TagsActions.fetchTags({ domain }));
    }
}
