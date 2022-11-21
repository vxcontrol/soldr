import { clone, NcFormProperty, NcformSchema, PropertyType } from '@soldr/shared';

export function localizeSchemaParams(configSchema: NcformSchema, configLocales: Record<string, any>, lang: string) {
    const schema = clone(configSchema) as NcformSchema;

    Object.keys(schema.properties).forEach((property) => {
        const ui = schema.properties[property].ui || ({} as NcFormProperty['ui']);

        const propertyLocale = configLocales[property] as Record<string, any>;

        ui.label = (propertyLocale[lang] || {}).title;
        if (
            schema.properties[property].type === PropertyType.ARRAY ||
            schema.properties[property].type === PropertyType.OBJECT
        ) {
            ui.legend = (propertyLocale[lang] || {}).description;
        } else {
            ui.description = (propertyLocale[lang] || {}).description;
        }
    });

    return schema;
}
