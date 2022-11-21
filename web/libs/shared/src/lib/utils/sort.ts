import { ModelsGroup, ModelsModuleAShort, ModelsPolicy } from '@soldr/api';

import { GridColumnFilterItem } from '../components';

export function sortModules(language: string) {
    return (a: ModelsModuleAShort, b: ModelsModuleAShort) =>
        a.locale.module[language].title.localeCompare(b.locale.module[language].title, 'en');
}

export function sortPolicies(language: string) {
    return (a: ModelsPolicy, b: ModelsPolicy) => a.info.name[language].localeCompare(b.info.name[language], 'en');
}

export function sortTags() {
    return (a: string, b: string) => a.localeCompare(b, 'en');
}

export function sortGroups(language: string) {
    return (a: ModelsGroup, b: ModelsGroup) => a.info.name[language].localeCompare(b.info.name[language], 'en');
}

export function sortGridColumnFilterItems() {
    return (a: GridColumnFilterItem, b: GridColumnFilterItem) => a.label.localeCompare(b.label, 'en');
}
