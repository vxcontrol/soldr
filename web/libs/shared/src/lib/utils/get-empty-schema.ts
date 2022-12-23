import { NcformSchema, PropertyType } from '@soldr/shared';

export function getEmptySchema(additionalProperties?: boolean): NcformSchema {
    const schema: NcformSchema = {
        type: PropertyType.OBJECT,
        properties: {},
        required: []
    };

    if (additionalProperties !== undefined) {
        schema.additionalProperties = additionalProperties;
    }

    return schema;
}
