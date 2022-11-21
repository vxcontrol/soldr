import { InjectionToken } from '@angular/core';

import { Permission } from '@soldr/api';
import { PageUrls, PermissionsService } from '@soldr/core';
import { ProxyPermission, ProxyPermissionPage } from '@soldr/shared';

export const PERMISSIONS_TOKEN = new InjectionToken<ProxyPermission | ProxyPermissionPage>('Proxy Permission');
export const PROXY_PERMISSION = (permissionsService: PermissionsService) =>
    new Proxy(
        {},
        {
            get(target: any, value: string): boolean {
                if (Object.keys(PageUrls).includes(value)) {
                    const pagePerm = PageUrls[value as keyof typeof PageUrls];

                    return permissionsService.hasAccessToPage(pagePerm);
                } else {
                    const perm = Permission[value as keyof typeof Permission];

                    return permissionsService.hasPermission(perm);
                }
            }
        }
    );
