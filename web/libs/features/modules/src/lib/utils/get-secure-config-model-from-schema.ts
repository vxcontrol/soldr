import { NcFormProperty, NcformSchema, PropertyType } from '@soldr/shared';

import { SecureConfigurationItem } from '../types';

const TAB_SIZE = 2;

export function getSecureConfigModelFromSchema(
    schema: NcformSchema | NcFormProperty,
    inline?: boolean
): SecureConfigurationItem[] {
    const model: SecureConfigurationItem[] = [];

    const make = (obj: NcformSchema | NcFormProperty, base: string, root?: SecureConfigurationItem) => {
        if (typeof obj !== 'object' || Array.isArray(obj)) {
            return;
        }
        if (typeof obj.properties === 'object' && !Array.isArray(obj.properties)) {
            const reqs = Array.isArray(obj.required) ? obj.required : [];

            for (const key of Object.keys(obj.properties)) {
                const name = base === '' ? key : `${base}.${key}`;
                const isRoot = base === '';
                const serverOnly = isRoot ? obj.properties[key].properties.server_only.value : root?.serverOnly;
                const type = isRoot ? obj.properties[key].properties.value.type : obj.properties[key].type;
                const value = isRoot ? obj.properties[key].properties.value : obj.properties[key];

                const item: SecureConfigurationItem = {
                    required: reqs.indexOf(key) !== -1,
                    serverOnly,
                    name,
                    type,
                    fields: JSON.stringify(
                        Object.keys(value)
                            .filter((k: string) => !['type', 'required'].includes(k) && (k !== 'properties' || inline))
                            .reduce((res: Record<string, any>, k: string) => {
                                res[k] = (obj.properties[key].properties.value as Record<string, any>)[k];

                                return res;
                            }, {} as Record<string, any>),
                        undefined,
                        TAB_SIZE
                    )
                };

                model.push(item);
                if (type === PropertyType.OBJECT && !inline) {
                    make(value, name, item);
                }
            }
        }
    };

    make(schema, '');

    return model;
}
