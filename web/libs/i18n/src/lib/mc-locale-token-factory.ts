import { inject, LOCALE_ID } from '@angular/core';

// NOTE: we need to use short locale name for MC_LOCALE_ID and MC_DATE_LOCALE tokens
// see MomentDateAdapter.setLocale and McDecimalPipe
export function mcLocaleTokensFactory(): string {
    const localeId = inject(LOCALE_ID); // берём выставленное заранее значение глобального ангулярного токена LOCALE_ID\

    return localeId.split('-')[0];
}
