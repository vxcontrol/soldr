import { Injectable } from '@angular/core';
import { CanActivateChild, UrlTree } from '@angular/router';
import { Observable, of } from 'rxjs';

import { LocationService, PermissionsService } from '@soldr/core';
import { getFeaturePath } from '@soldr/shared';

@Injectable({ providedIn: 'root' })
export class ErrorsGuard implements CanActivateChild {
    constructor(private locationService: LocationService, private permissionsService: PermissionsService) {}

    canActivateChild(): Observable<boolean | UrlTree> | Promise<boolean | UrlTree> | boolean | UrlTree {
        const path = getFeaturePath(window.location.pathname);
        if (!this.permissionsService.isMatchAnyPath(`/${path}`) && path) {
            this.locationService.redirectToFirstAvailablePage();

            return of(false);
        }

        return of(true);
    }
}
