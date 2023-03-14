import { Injectable } from '@angular/core';
import { CanActivateChild, Router, UrlTree } from '@angular/router';
import { combineLatest, filter, first, map, Observable } from 'rxjs';

import { SharedFacade } from '@soldr/store/shared';

@Injectable({
    providedIn: 'root'
})
export class UnauthorizedGuard implements CanActivateChild {
    constructor(private sharedFacade: SharedFacade, private router: Router) {}

    canActivateChild(): Observable<boolean | UrlTree> | Promise<boolean | UrlTree> | boolean | UrlTree {
        this.sharedFacade.fetchInfo();

        return combineLatest([this.sharedFacade.selectInfo(), this.sharedFacade.selectIsInfoLoaded()]).pipe(
            filter(([info, loaded]) => !!info && loaded),
            first(),
            map(([info]) => {
                if (info?.type === 'user') {
                    return this.router.parseUrl('/');
                }

                return true;
            })
        );
    }
}
