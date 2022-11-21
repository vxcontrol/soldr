import { Injectable } from '@angular/core';
import { TranslocoMissingHandler } from '@ngneat/transloco';

@Injectable({ providedIn: 'root' })
export class MissingKeyHandler implements TranslocoMissingHandler {
    handle(key: string) {
        console.warn('Missing translation key: ', key);

        return '';
    }
}
