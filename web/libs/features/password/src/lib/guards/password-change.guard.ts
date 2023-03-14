import { Injectable } from '@angular/core';
import { ActivatedRouteSnapshot, CanActivate, Router, RouterStateSnapshot } from '@angular/router';
import { first, map, Observable } from 'rxjs';

import { LocationService } from '@soldr/core';
import { SharedFacade } from '@soldr/store/shared';

@Injectable({ providedIn: 'root' })
export class PasswordChangeGuard implements CanActivate {
    constructor(private locationService: LocationService, private router: Router, private sharedFacade: SharedFacade) {}

    canActivate(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<boolean> {
        return this.sharedFacade.isPasswordChangeRequired$.pipe(
            first(),
            map((isPasswordChangeRequired) => {
                if (isPasswordChangeRequired === false || !this.router.routerState.snapshot.url.includes('login')) {
                    this.locationService.redirectToFirstAvailablePage();

                    return false;
                }

                return true;
            })
        );
    }
}
