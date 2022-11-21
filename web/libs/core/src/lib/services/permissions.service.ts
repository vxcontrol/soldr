import { Injectable } from '@angular/core';

import { Permission, PublicInfo } from '@soldr/api';
import { SharedFacade } from '@soldr/store/shared';

import { PageUrls } from '../types';

@Injectable({
    providedIn: 'root'
})
export class PermissionsService {
    info: PublicInfo;
    pages = Object.values(PageUrls);

    readonly urlPermissions: Record<string, Permission[]> = {
        [PageUrls.Agents]: [Permission.ViewAgents],
        [PageUrls.Policies]: [Permission.ViewPolicies],
        [PageUrls.Groups]: [Permission.ViewGroups],
        [PageUrls.Modules]: [Permission.ViewModules],
        [PageUrls.Downloads]: [Permission.DownloadsAgents]
    };

    constructor(private sharedFacade: SharedFacade) {
        sharedFacade.selectInfo().subscribe((info) => (this.info = info));
    }

    get isEmptyPermission(): boolean {
        return this.info?.privileges?.length === 0;
    }

    isMatchAnyPath(page: string) {
        return this.pages.find((path) => path === page);
    }

    hasPermission(permission: Permission) {
        return this.info?.privileges?.indexOf(permission) !== -1;
    }

    hasAccessToPage(page: string) {
        return this.urlPermissions[page].every((perm) => this.hasPermission(perm));
    }

    getFirstAvailablePage(): string {
        return this.pages.find((page) => this.hasAccessToPage(page));
    }
}
