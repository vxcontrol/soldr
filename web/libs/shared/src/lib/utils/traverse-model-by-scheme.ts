import { NcFormProperty, PropertyType } from '../types';

export function traverseModelByScheme(
    schema: NcFormProperty,
    model: Record<string, any>,
    callback: (parent: NcFormProperty, parentModel: Record<string, any>, propName: string) => void
) {
    if (schema.type === PropertyType.OBJECT) {
        for (const [propName, prop] of Object.entries(schema.properties || {})) {
            if (prop.type === PropertyType.OBJECT) {
                traverseModelByScheme(prop, model[propName] as Record<string, any>, callback);
            } else {
                callback(schema, model, propName);
            }
        }
    }
}
