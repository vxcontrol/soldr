import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { EffectsModule } from '@ngrx/effects';
import { StoreModule } from '@ngrx/store';

import { GroupsEffects } from './groups.effects';
import * as fromGroups from './groups.reducer';

@NgModule({
    imports: [
        CommonModule,
        StoreModule.forFeature(fromGroups.groupsFeatureKey, fromGroups.reducer),
        EffectsModule.forFeature([GroupsEffects])
    ]
})
export class StoreGroupsModule {}
