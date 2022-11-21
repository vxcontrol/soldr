import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';

import { SharedModule } from '@soldr/shared';

import { ModuleInteractivePartComponent, ModulePageComponent } from './components';

const components = [ModuleInteractivePartComponent, ModulePageComponent];

@NgModule({
    imports: [CommonModule, SharedModule],
    declarations: [...components],
    exports: [...components]
})
export class FeaturesModulesInteractivityModule {}
