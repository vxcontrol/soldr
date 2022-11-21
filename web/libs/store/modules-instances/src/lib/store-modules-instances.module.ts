import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { EffectsModule } from '@ngrx/effects';
import { StoreModule } from '@ngrx/store';

import { ModulesInstancesEffects } from './modules-instances.effects';
import * as fromModulesInstances from './modules-instances.reducer';

@NgModule({
    imports: [
        CommonModule,
        StoreModule.forFeature(fromModulesInstances.modulesInstancesFeatureKey, fromModulesInstances.reducer),
        EffectsModule.forFeature([ModulesInstancesEffects])
    ]
})
export class StoreModulesInstancesModule {}
