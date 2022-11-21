import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { EffectsModule } from '@ngrx/effects';
import { StoreModule } from '@ngrx/store';

import { TagsEffects } from './tags.effects';
import * as fromTags from './tags.reducer';

@NgModule({
    imports: [
        CommonModule,
        StoreModule.forFeature(fromTags.tagsFeatureKey, fromTags.reducer),
        EffectsModule.forFeature([TagsEffects])
    ]
})
export class StoreTagsModule {}
