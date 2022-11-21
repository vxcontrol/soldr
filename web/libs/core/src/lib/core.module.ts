import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';

import { PERMISSIONS_TOKEN, PermissionsService, PROXY_PERMISSION } from './services';

@NgModule({
    imports: [CommonModule],
    providers: [{ provide: PERMISSIONS_TOKEN, useFactory: PROXY_PERMISSION, deps: [PermissionsService] }]
})
export class CoreModule {}
