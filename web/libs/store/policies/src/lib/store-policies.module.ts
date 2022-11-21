import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { EffectsModule } from '@ngrx/effects';
import { StoreModule } from '@ngrx/store';

import { PoliciesEffects } from './policies.effects';
import * as fromPolicies from './policies.reducer';

@NgModule({
    imports: [
        CommonModule,
        StoreModule.forFeature(fromPolicies.policiesFeatureKey, fromPolicies.reducer),
        EffectsModule.forFeature([PoliciesEffects])
    ]
})
export class StorePoliciesModule {}
