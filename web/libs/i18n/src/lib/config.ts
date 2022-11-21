import { translocoConfig } from '@ngneat/transloco';

import { environment } from '@soldr/environments';

export const LOCALES = {
    ru_RU: 'ru-RU',
    en_US: 'en-US'
};

export const LANGUAGES = {
    ru: 'ru',
    en: 'en'
};

export type DataLanguage = 'ru' | 'en';

export const AVAILABLE_LOCALES = [LOCALES.ru_RU, LOCALES.en_US];
export const DEFAULT_LOCALE = LOCALES.en_US;
export const FALLBACK_LOCALE = LOCALES.en_US;

export const defaultTranslocoConfig = translocoConfig({
    availableLangs: AVAILABLE_LOCALES,
    defaultLang: DEFAULT_LOCALE,
    fallbackLang: FALLBACK_LOCALE,
    reRenderOnLangChange: false,
    prodMode: environment.production
});
