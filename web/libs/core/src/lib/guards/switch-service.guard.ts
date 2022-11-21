import { Injectable } from '@angular/core';
import { ActivatedRouteSnapshot, CanActivate, Router, RouterStateSnapshot, UrlTree } from '@angular/router';
import { TranslocoService } from '@ngneat/transloco';
import { McToastService } from '@ptsecurity/mosaic/toast';
import { filter, first, map, Observable } from 'rxjs';

import { ModelsService, PublicService, SuccessResponse } from '@soldr/api';
import { getFeaturePath } from '@soldr/shared';
import { SharedFacade } from '@soldr/store/shared';

import { PermissionsService } from '../services';

@Injectable({
    providedIn: 'root'
})
export class SwitchServiceGuard implements CanActivate {
    constructor(
        private permissionsService: PermissionsService,
        private publicService: PublicService,
        private router: Router,
        private sharedFacade: SharedFacade,
        private toastService: McToastService,
        private translocoService: TranslocoService
    ) {}

    canActivate(
        route: ActivatedRouteSnapshot,
        state: RouterStateSnapshot
    ): Observable<boolean | UrlTree> | Promise<boolean | UrlTree> | boolean | UrlTree {
        return this.sharedFacade.selectInfo().pipe(
            filter((info) => info?.type === 'user'),
            first(),
            map((info) => {
                const featureRoute = getFeaturePath(state.url);
                if (!localStorage.getItem('server_hash')) {
                    localStorage.setItem('server_hash', info.service.hash);
                }
                if ((route && route.params.service_hash !== info.service.hash) || !featureRoute) {
                    const foundSever = info.services.find(({ hash }) => hash === route.params.service_hash);
                    const lastSelected = localStorage.getItem('server_hash');
                    const foundLastSelected = info.services.find(({ hash }) => hash === lastSelected);

                    const nextUrl =
                        route.url[0]?.path && this.permissionsService.isMatchAnyPath(`/${route.url[0]?.path}`)
                            ? `/${route.url.join('/')}`
                            : featureRoute && this.permissionsService.isMatchAnyPath(`/${featureRoute}`)
                            ? `/${featureRoute}`
                            : this.permissionsService.getFirstAvailablePage();

                    const queryParams = route.queryParams;

                    if (foundSever && route.params.service_hash !== info.service.hash) {
                        this.publicService.switchService(foundSever.hash).subscribe({
                            next: (response: SuccessResponse<ModelsService>) => {
                                const service = response.data;
                                localStorage.setItem('server_hash', service.hash);

                                window.location.reload();
                            },
                            error: () => {
                                this.router.navigateByUrl(`/services/${info.service.hash}${nextUrl}`, {
                                    replaceUrl: true
                                });
                                this.toastService.show({
                                    style: 'error',
                                    title: this.translocoService.translate(
                                        'shared.Shared.SwitchService.ErrorText.ChangeServiceTitle'
                                    ),
                                    caption: this.translocoService.translate(
                                        'shared.Shared.SwitchService.ErrorText.ChangeServiceCaption'
                                    )
                                });
                            }
                        });
                    } else if (foundLastSelected) {
                        this.router.navigate([`/services/${foundLastSelected.hash}${nextUrl}`], {
                            queryParams
                        });
                    } else {
                        this.router.navigate([`/services/${info.service.hash}${nextUrl}`], {
                            queryParams
                        });
                    }
                }

                return true;
            })
        );
    }
}
