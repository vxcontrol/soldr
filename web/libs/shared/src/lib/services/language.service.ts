import { Injectable } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';
import { map } from 'rxjs';

import { ListQueryLang } from '@soldr/api';

@Injectable({
    providedIn: 'root'
})
export class LanguageService {
    constructor(private transloco: TranslocoService) {}

    current$ = this.transloco.langChanges$.pipe(map(this.localeToLang));

    get lang() {
        return this.localeToLang(this.transloco.getActiveLang()) as ListQueryLang;
    }

    private localeToLang(value: string) {
        return value.split('-')[0];
    }
}
