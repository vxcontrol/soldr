import { Injectable } from '@angular/core';
import { CanActivateChild, Router, UrlTree } from '@angular/router';
import { combineLatest, filter, first, map, Observable } from 'rxjs';

import { SharedFacade } from '@soldr/store/shared';

import { PermissionsService } from '../services';
import { PASSWORD_CHANGE_PAGE } from '../types';

@Injectable({
    providedIn: 'root'
})
export class AuthorizedGuard implements CanActivateChild {
    constructor(
        private permissionsService: PermissionsService,
        private router: Router,
        private sharedFacade: SharedFacade
    ) {}

    canActivateChild(): Observable<boolean | UrlTree> | Promise<boolean | UrlTree> | boolean | UrlTree {
        this.sharedFacade.fetchInfo();

        return combineLatest([this.sharedFacade.selectInfo(), this.sharedFacade.selectIsInfoLoaded()]).pipe(
            filter(([info, loaded]) => !!info && loaded),
            first(),
            map(([info]) => {
                if (info?.type !== 'user') {
                    const nextUrl = window.location.href
                        .replace(window.location.origin, '')
                        .replace(PASSWORD_CHANGE_PAGE, '');
                    const firstAvailablePage = this.permissionsService.getFirstAvailablePage();

                    return this.router.createUrlTree(['/login'], {
                        queryParams: { nextUrl: nextUrl || firstAvailablePage }
                    });
                }

                return true;
            })
        );
    }
}
