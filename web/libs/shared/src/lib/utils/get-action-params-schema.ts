import { clone, EntityModule, NcformSchema } from '@soldr/shared';

export function getActionParamsSchema(module: EntityModule, actionName: string): NcformSchema {
    let paramsSchema = clone(
        (module.action_config_schema as NcformSchema).properties[actionName].allOf[1]
    ) as NcformSchema;

    delete paramsSchema.properties.fields;
    delete paramsSchema.properties.priority;

    paramsSchema = JSON.parse(
        JSON.stringify(paramsSchema).replace(new RegExp(`\\$root\\.${actionName}`, 'g'), '$root')
    ) as NcformSchema;

    return paramsSchema;
}
