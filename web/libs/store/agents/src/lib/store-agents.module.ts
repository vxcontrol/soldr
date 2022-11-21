import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';

import { StoreAgentCardModule } from './card/store-agent-card.module';
import { StoreAgentListModule } from './list/store-agent-list.module';

@NgModule({
    imports: [CommonModule, StoreAgentCardModule, StoreAgentListModule]
})
export class StoreAgentsModule {}
