import { NcFormProperty, NcformSchema, PropertyType } from '@soldr/shared';

import { ConfigurationItem } from '../types';

const TAB_SIZE = 2;

export function getModelFromSchema(schema: NcformSchema | NcFormProperty, inline?: boolean): ConfigurationItem[] {
    const model: ConfigurationItem[] = [];

    const make = (obj: NcformSchema | NcFormProperty, base: string) => {
        if (typeof obj !== 'object' || Array.isArray(obj)) {
            return;
        }
        if (typeof obj.properties === 'object' && !Array.isArray(obj.properties)) {
            const reqs = Array.isArray(obj.required) ? obj.required : [];

            for (const key of Object.keys(obj.properties)) {
                const name = base === '' ? key : `${base}.${key}`;
                model.push({
                    required: reqs.indexOf(key) !== -1,
                    name,
                    type: obj.properties[key].type,
                    fields: JSON.stringify(
                        Object.keys(obj.properties[key])
                            .filter((k: string) => k !== 'type' && (k !== 'properties' || inline) && k !== 'required')
                            .reduce((res: Record<string, any>, k: string) => {
                                res[k] = (obj.properties[key] as Record<string, any>)[k];

                                return res;
                            }, {} as Record<string, any>),
                        undefined,
                        TAB_SIZE
                    )
                });
                if (obj.properties[key].type === PropertyType.OBJECT && !inline) {
                    make(obj.properties[key], name);
                }
            }
        }
    };

    make(schema, '');

    return model;
}
