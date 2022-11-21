import { Injectable } from '@angular/core';
import { ActivatedRouteSnapshot, CanActivate } from '@angular/router';
import { Translation, TranslocoService } from '@ngneat/transloco';
import { forkJoin, map, Observable, of } from 'rxjs';

import { DEFAULT_LOCALE } from './config';

const DEFAULT_SCOPES = ['shared', 'common'];

@Injectable({ providedIn: 'root' })
export class LazyLoadTranslationsGuard implements CanActivate {
    private loadedScopes = new Set<string>();
    private scopeNames: string[] = [];

    constructor(private transloco: TranslocoService) {
        transloco.setActiveLang(localStorage.getItem('locale') || DEFAULT_LOCALE);
    }

    canActivate(route: ActivatedRouteSnapshot): Observable<boolean> {
        const scope = route.data.scope || [];
        this.scopeNames = [...DEFAULT_SCOPES, ...scope];
        const observables: Observable<Translation>[] = [];

        if (this.scopeNames.every((scopeItem) => this.loadedScopes.has(scopeItem))) {
            return of(true);
        }

        this.scopeNames.forEach((scopeItem) => {
            if (!this.loadedScopes.has(scopeItem)) {
                const path = `${scopeItem}/${this.transloco.getActiveLang()}`;
                observables.push(this.transloco.load(path));
                this.loadedScopes.add(scopeItem);
            }
        });

        return forkJoin(observables).pipe(map(() => true));
    }
}
