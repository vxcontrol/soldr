import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';

import { StoreModuleEditModule } from './edit/store-module-edit.module';
import { StoreModuleListModule } from './list/store-module-list.module';

@NgModule({
    imports: [CommonModule, StoreModuleListModule, StoreModuleEditModule]
})
export class StoreModulesModule {}
