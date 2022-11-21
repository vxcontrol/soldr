import { Inject, Injectable } from '@angular/core';
import { ActivatedRouteSnapshot, CanActivate, Router, UrlTree } from '@angular/router';
import { Observable } from 'rxjs';

import { LocationService, PERMISSIONS_TOKEN } from '@soldr/core';
import { ProxyPermission } from '@soldr/shared';

@Injectable({ providedIn: 'root' })
export class ModulesGuard implements CanActivate {
    constructor(
        private router: Router,
        private locationService: LocationService,
        @Inject(PERMISSIONS_TOKEN) private permitted: ProxyPermission
    ) {}

    canActivate(
        route: ActivatedRouteSnapshot
    ): Observable<boolean | UrlTree> | Promise<boolean | UrlTree> | boolean | UrlTree {
        if (route.data.actions.find((action: string) => action === 'view') && !this.permitted.ViewPolicies) {
            this.locationService.redirectToErrorPage();

            return false;
        }

        if (route.data.actions.find((action: string) => action === 'edit') && !this.permitted.EditModules) {
            this.locationService.redirectToErrorPage();

            return false;
        }

        return true;
    }
}
