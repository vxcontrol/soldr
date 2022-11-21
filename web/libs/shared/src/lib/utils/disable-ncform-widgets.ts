import { NcFormProperty, NcformSchema, PropertyType } from '../types';

export function disableNcformWidgets(v: Record<string, any>) {
    if (typeof v !== 'object') {
        return;
    }

    if (
        [
            PropertyType.BOOLEAN,
            PropertyType.STRING,
            PropertyType.NUMBER,
            PropertyType.INTEGER,
            PropertyType.OBJECT
        ].includes(v.type as PropertyType)
    ) {
        if (!v.ui) {
            v.ui = {};
        }
        v.ui.disabled = true;
    }

    if (v.type === PropertyType.ARRAY) {
        if (!v.ui) {
            v.ui = {};
        }

        if (!v.ui.widgetConfig) {
            v.ui.widgetConfig = {};
        }

        v.ui.disabled = true;
        v.ui.widgetConfig.disableAdd = true;
        v.ui.widgetConfig.disableDel = true;
    }

    if (v.type === PropertyType.OBJECT) {
        Object.keys((v as NcFormProperty).properties).forEach((property) => {
            if (!v.properties[property].type) {
                v.properties[property].type = PropertyType.STRING;
            }
            disableNcformWidgets(v.properties[property] as Record<string, any>);
        });

        Object.keys((v as NcformSchema).definitions || {}).forEach((definition) => {
            disableNcformWidgets(v.definitions[definition] as Record<string, any>);
        });
    } else {
        Object.keys(v).forEach((property) => disableNcformWidgets(v[property] as Record<string, any>));
    }
}
