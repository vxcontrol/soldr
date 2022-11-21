import { PropertyType } from '@soldr/shared';

export function getDefaultValueByType(type: PropertyType) {
    switch (type) {
        case PropertyType.OBJECT:
            return {};
        case PropertyType.ARRAY:
            return [];
        case PropertyType.BOOLEAN:
            return false;
        case PropertyType.INTEGER:
        case PropertyType.NUMBER:
            return 0;
        default:
            return '';
    }
}
