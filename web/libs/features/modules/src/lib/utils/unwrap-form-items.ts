import { clone, NcformSchema } from '@soldr/shared';

export function unwrapFormItems(schema: NcformSchema, unwrappedNames: string[] = []) {
    const value = clone(schema) as NcformSchema;

    for (const [name, item] of Object.entries(value.properties)) {
        if (unwrappedNames.includes(name)) {
            if (!item.allOf) {
                item.allOf = [];
            }

            item.allOf.push({
                ui: {
                    widgetConfig: {
                        collapsed: 'dx: false'
                    }
                }
            });
        }
    }

    return value;
}
