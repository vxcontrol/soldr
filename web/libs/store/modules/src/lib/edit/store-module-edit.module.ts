import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { EffectsModule } from '@ngrx/effects';
import { StoreModule } from '@ngrx/store';

import { ModuleEditEffects } from './module-edit.effects';
import * as fromModuleEdit from './module-edit.reducer';

@NgModule({
    imports: [
        CommonModule,
        StoreModule.forFeature(fromModuleEdit.moduleEditFeatureKey, fromModuleEdit.reducer),
        EffectsModule.forFeature([ModuleEditEffects])
    ]
})
export class StoreModuleEditModule {}
