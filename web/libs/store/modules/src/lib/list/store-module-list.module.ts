import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { EffectsModule } from '@ngrx/effects';
import { StoreModule } from '@ngrx/store';

import { ModuleListEffects } from './module-list.effects';
import * as fromModuleList from './module-list.reducer';

@NgModule({
    imports: [
        CommonModule,
        StoreModule.forFeature(fromModuleList.moduleListFeatureKey, fromModuleList.reducer),
        EffectsModule.forFeature([ModuleListEffects])
    ]
})
export class StoreModuleListModule {}
