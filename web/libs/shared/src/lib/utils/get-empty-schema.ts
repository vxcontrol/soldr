import { NcformSchema, PropertyType } from '@soldr/shared';

export function getEmptySchema(): NcformSchema {
    return {
        type: PropertyType.OBJECT,
        properties: {},
        required: []
    };
}
