import { DateTime } from 'luxon';

import { ModelsChangelogVersion } from '@soldr/api';
import { CHANGELOG_DATE_FORMAT_EN, CHANGELOG_DATE_FORMAT_RU } from '@soldr/features/modules';
import { LANGUAGES } from '@soldr/i18n';

import { ChangelogVersionRecord } from '../types';

export function getChangelogVersionModel(model: ChangelogVersionRecord) {
    return {
        [LANGUAGES.ru]: {
            date: DateTime.fromISO(model.date).toFormat(CHANGELOG_DATE_FORMAT_RU),
            title: model.locales.ru.title,
            description: model.locales.ru.description
        },
        [LANGUAGES.en]: {
            date: DateTime.fromISO(model.date).toFormat(CHANGELOG_DATE_FORMAT_EN),
            title: model.locales.en.title,
            description: model.locales.en.description
        }
    } as ModelsChangelogVersion;
}
