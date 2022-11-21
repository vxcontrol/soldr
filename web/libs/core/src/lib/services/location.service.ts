import { Injectable } from '@angular/core';
import { Router } from '@angular/router';

import { PermissionsService } from './permissions.service';

@Injectable({ providedIn: 'root' })
export class LocationService {
    constructor(private permissionsService: PermissionsService, private router: Router) {}

    redirectToFirstAvailablePage() {
        const firstAvailablePage = this.permissionsService.getFirstAvailablePage();
        this.router.navigateByUrl(firstAvailablePage);
    }

    redirectToErrorPage() {
        this.router.navigateByUrl('/errors', { skipLocationChange: true });
    }
}
