import { NcFormProperty, NcformSchema, PropertyType } from '@soldr/shared';

import { SecureConfigurationItem } from '../types';

export function getSecureConfigSchemaFromModel(
    model: SecureConfigurationItem[],
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

    model.forEach((propertyModel: SecureConfigurationItem) => {
        let obj: NcformSchema | NcFormProperty = schema;
        let parent: NcformSchema | NcFormProperty = schema;
        const name = propertyModel.name === undefined ? '' : propertyModel.name;
        const path = !inline ? name.split('.') : [name];
        const field = path.slice(-1)[0];
        const isRoot = path.length === 1;

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
        if (propertyModel.required) {
            if (isRoot) {
                obj.required.push(field);
            } else {
                obj.properties?.value.required.push(field);
            }
        }

        let fieldObject = isRoot ? obj.properties[field] : obj.properties.value.properties?.[field];

        if (fieldObject === undefined) {
            fieldObject = isRoot
                ? {
                      type: PropertyType.OBJECT,
                      properties: {
                          server_only: {
                              type: PropertyType.BOOLEAN,
                              value: propertyModel.serverOnly,
                              ui: {
                                  hidden: true
                              },
                              enum: [propertyModel.serverOnly]
                          },
                          value: {
                              type: PropertyType.STRING
                          }
                      },
                      required: ['server_only', 'value']
                  }
                : ({
                      type: PropertyType.OBJECT,
                      properties: {},
                      required: []
                  } as NcFormProperty);

            if (isRoot) {
                obj.properties[field] = fieldObject;
            } else {
                if (!obj.properties.value.properties) {
                    obj.properties.value.properties = {};
                }
                obj.properties.value.properties[field] = fieldObject;
            }
        }
        const type = isRoot ? fieldObject.properties.value.type : fieldObject.type;
        const valueObject = isRoot ? fieldObject.properties.value : fieldObject;

        if (type !== PropertyType.NONE) {
            valueObject.type = propertyModel.type as PropertyType;

            if (propertyModel.type === PropertyType.OBJECT) {
                valueObject.properties = {};
            }

            if (propertyModel.type === PropertyType.ARRAY) {
                valueObject.items = [];
            }
        }
        if (type === PropertyType.OBJECT) {
            if (typeof valueObject.properties !== 'object') {
                valueObject.properties = {};
            }
            if (typeof fieldObject.required !== 'object' || !Array.isArray(fieldObject.required)) {
                fieldObject.required = [];
            }
        }
        try {
            if (isRoot) {
                Object.assign(fieldObject.properties.value, JSON.parse(propertyModel.fields));
            } else {
                Object.assign(fieldObject, JSON.parse(propertyModel.fields));
            }
        } catch (e) {}
    });

    return schema;
}
