import { Injectable } from '@angular/core';
import { ActivatedRouteSnapshot, CanActivateChild } from '@angular/router';

import { LocationService, PermissionsService } from '@soldr/core';

@Injectable({
    providedIn: 'root'
})
export class PermissionsGuard implements CanActivateChild {
    constructor(private locationService: LocationService, private permissionsService: PermissionsService) {}

    canActivateChild(childRoute: ActivatedRouteSnapshot): boolean {
        const route = childRoute.url[0].path;
        if (!this.permissionsService.hasAccessToPage(`/${route}`)) {
            this.locationService.redirectToErrorPage();

            return false;
        }

        return true;
    }
}
