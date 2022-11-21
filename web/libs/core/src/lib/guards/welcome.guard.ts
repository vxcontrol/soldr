import { Injectable } from '@angular/core';
import { CanActivate, Router, UrlTree } from '@angular/router';
import { combineLatest, filter, first, map, Observable } from 'rxjs';

import { LocationService, PermissionsService } from '@soldr/core';
import { SharedFacade } from '@soldr/store/shared';

@Injectable({ providedIn: 'root' })
export class WelcomeGuard implements CanActivate {
    constructor(
        private locationService: LocationService,
        private permissionsService: PermissionsService,
        private router: Router,
        private sharedFacade: SharedFacade
    ) {}

    canActivate(): Observable<boolean | UrlTree> | Promise<boolean | UrlTree> | boolean | UrlTree {
        this.sharedFacade.fetchInfo();

        return combineLatest([this.sharedFacade.selectInfo(), this.sharedFacade.selectIsInfoLoaded()]).pipe(
            filter(([info, loaded]) => Boolean(info) && loaded),
            first(),
            map(([info]) => {
                if (info?.type === 'user') {
                    if (this.permissionsService.isEmptyPermission) {
                        setTimeout(() => this.locationService.redirectToErrorPage());

                        return false;
                    }

                    this.locationService.redirectToFirstAvailablePage();

                    return false;
                }

                this.router.navigateByUrl('/login');

                return false;
            })
        );
    }
}
