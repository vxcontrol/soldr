import { Injectable } from '@angular/core';
import { CanDeactivate, UrlTree } from '@angular/router';
import { Observable, of } from 'rxjs';

import { ModalInfoService } from '../services';
import { CanLeavePage } from '../types';

@Injectable({
    providedIn: 'root'
})
export class CanLeavePageGuard implements CanDeactivate<CanLeavePage> {
    constructor(private modalInfoService: ModalInfoService) {}

    canDeactivate(
        component: CanLeavePage
    ): Observable<boolean | UrlTree> | Promise<boolean | UrlTree> | boolean | UrlTree {
        if (!component.canLeavePage) {
            return this.modalInfoService.openUnsavedChangesModal();
        }

        return of(true);
    }
}
