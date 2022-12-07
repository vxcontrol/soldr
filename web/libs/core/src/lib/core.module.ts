import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';

import {
    PERMISSIONS_TOKEN,
    PermissionsService,
    PROXY_PERMISSION,
    PROXY_THEME_SERVICE,
    THEME_TOKENS,
    ThemeService
} from './services';

@NgModule({
    imports: [CommonModule],
    providers: [
        { provide: PERMISSIONS_TOKEN, useFactory: PROXY_PERMISSION, deps: [PermissionsService] },
        {
            provide: THEME_TOKENS,
            useFactory: PROXY_THEME_SERVICE,
            deps: [ThemeService]
        }
    ]
})
export class CoreModule {}
