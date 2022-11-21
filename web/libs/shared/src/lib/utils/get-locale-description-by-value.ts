import { ModelsModuleLocaleDesc } from '@soldr/api';
import { LANGUAGES } from '@soldr/i18n';

export function getLocaleDescriptionByValue(value: string): ModelsModuleLocaleDesc {
    return Object.keys(LANGUAGES).reduce((acc, lang) => ({ ...acc, [lang]: { title: value, description: value } }), {});
}
