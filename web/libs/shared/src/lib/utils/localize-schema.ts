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
        let plainSchema = JSON.stringify(schema);
        let match;
        while ((match = REGEX_PARSE_SCHEMA_LOCALIZATION.exec(plainSchema))) {
            // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
            plainSchema = plainSchema.replace(match[1], additionalKeys[match[1]][lang]);
        }

        return JSON.parse(plainSchema);
    }

    return schema;
};
