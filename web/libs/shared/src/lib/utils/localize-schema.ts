import { injectorInstance } from '@soldr/api';

import { REGEX_PARSE_SCHEMA_LOCALIZATION } from '../constants';
import { LanguageService } from '../services';

export const localizeSchemaAdditionalKeys = (
    schema: any,
    additionalKeys: Record<string, Record<string, string>>,
    languageService = injectorInstance.get<LanguageService>(LanguageService)
) => {
    if (Object.keys(additionalKeys || {}).length) {
        const lang = languageService.lang;
        const schemaKeys = new Set<string>();

        let plainSchema = JSON.stringify(schema);
        let match;

        while ((match = REGEX_PARSE_SCHEMA_LOCALIZATION.exec(plainSchema)) !== null) {
            schemaKeys.add(String(match[1]));
        }

        schemaKeys.forEach((key) => (plainSchema = plainSchema.replace(key, additionalKeys[key]?.[lang] || '')));

        return JSON.parse(plainSchema);
    }

    return schema;
};
