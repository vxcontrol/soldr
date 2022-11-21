import { clone, EntityModule, NcformSchema } from '@soldr/shared';

export function getEventParamsSchema(module: EntityModule, eventName: string): NcformSchema {
    let paramsSchema = clone(module.event_config_schema.properties[eventName].allOf[1]) as NcformSchema;

    delete paramsSchema.properties.fields;

    paramsSchema = JSON.parse(
        JSON.stringify(paramsSchema).replace(new RegExp(`\\$root\\.${eventName}`, 'g'), '$root')
    ) as NcformSchema;

    return paramsSchema;
}
