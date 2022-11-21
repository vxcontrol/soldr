import { NcFormProperty, NcformSchema, PropertyType } from '@soldr/shared';

import { ConfigurationItem } from '../types';

export function getSchemaFromModel(
    model: ConfigurationItem[],
    additionalProperties?: boolean,
    inline?: boolean
): NcformSchema {
    const schema: NcformSchema = {
        type: PropertyType.OBJECT,
        properties: {},
        required: []
    };

    if (additionalProperties !== undefined) {
        schema.additionalProperties = additionalProperties;
    }
    if (model === undefined) {
        return schema;
    }

    model.forEach((data: ConfigurationItem) => {
        let obj: NcformSchema | NcFormProperty = schema;
        let parent: NcformSchema | NcFormProperty = schema;
        const name = data.name === undefined ? '' : data.name;
        const path = !inline ? name.split('.') : [name];
        const field = path.slice(-1)[0];

        path.slice(0, -1).forEach((fieldName: string) => {
            obj = parent.properties[fieldName];
            if (typeof obj !== 'object' || Array.isArray(obj) || obj.type !== PropertyType.OBJECT) {
                obj = {
                    type: PropertyType.OBJECT,
                    properties: {},
                    required: []
                } as NcFormProperty;
                parent.properties[fieldName] = obj;
            }
            parent = obj;
        });
        if (data.required) {
            obj.required.push(field);
        }
        let fobj = obj.properties[field];
        if (fobj === undefined) {
            obj.properties[field] = fobj = {};
        }
        if (fobj.type !== PropertyType.NONE) {
            Object.assign(fobj, { type: data.type });
        }
        if (fobj.type === 'object') {
            if (typeof fobj.properties !== 'object' || Array.isArray(['properties'])) {
                fobj.properties = {};
            }
            if (typeof fobj.required !== 'object' || !Array.isArray(['required'])) {
                fobj.required = [];
            }
        }
        try {
            Object.assign(fobj, JSON.parse(data.fields));
        } catch (e) {}
    });

    return schema;
}
