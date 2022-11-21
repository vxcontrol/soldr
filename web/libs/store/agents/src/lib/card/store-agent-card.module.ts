import { NgModule } from '@angular/core';
import { EffectsModule } from '@ngrx/effects';
import { StoreModule } from '@ngrx/store';

import { AgentCardEffects } from './agent-card.effects';
import * as fromAgentCard from './agent-card.reducer';

@NgModule({
    imports: [
        StoreModule.forFeature(fromAgentCard.agentCardFeatureKey, fromAgentCard.reducer),
        EffectsModule.forFeature([AgentCardEffects])
    ]
})
export class StoreAgentCardModule {}
