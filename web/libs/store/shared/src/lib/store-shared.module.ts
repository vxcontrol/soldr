import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { EffectsModule } from '@ngrx/effects';
import { StoreModule } from '@ngrx/store';

import { SharedEffects } from './shared.effects';
import * as fromShared from './shared.reducer';

@NgModule({
    imports: [
        CommonModule,
        StoreModule.forFeature(fromShared.sharedFeatureKey, fromShared.reducer),
        EffectsModule.forFeature([SharedEffects])
    ]
})
export class StoreSharedModule {}
