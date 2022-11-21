import { DateTime } from 'luxon';

import { ModelsChangelogVersion } from '@soldr/api';
import { CHANGELOG_DATE_FORMAT_EN } from '@soldr/features/modules';
import { LANGUAGES } from '@soldr/i18n';

import { ChangelogVersionRecord } from '../types';

export function getChangelogVersionRecordFormModel(version: string, model: ModelsChangelogVersion) {
    const date = DateTime.fromFormat(model.en.date, CHANGELOG_DATE_FORMAT_EN).toISODate();

    return {
        date,
        version,
        locales: {
            ru: {
                title: model[LANGUAGES.ru].title,
                description: model[LANGUAGES.ru].description
            },
            en: {
                title: model[LANGUAGES.en].title,
                description: model[LANGUAGES.en].description
            }
        }
    } as ChangelogVersionRecord;
}
