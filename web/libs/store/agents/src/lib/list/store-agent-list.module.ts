import { NgModule } from '@angular/core';
import { EffectsModule } from '@ngrx/effects';
import { StoreModule } from '@ngrx/store';

import { AgentListEffects } from './agent-list.effects';
import * as fromAgentList from './agent-list.reducer';

@NgModule({
    imports: [
        StoreModule.forFeature(fromAgentList.agentListFeatureKey, fromAgentList.reducer),
        EffectsModule.forFeature([AgentListEffects])
    ]
})
export class StoreAgentListModule {}
