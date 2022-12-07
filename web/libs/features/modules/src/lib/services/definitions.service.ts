import { Injectable } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';

import { clone, definitionsSchema, NcFormProperty, replaceByProperties } from '@soldr/shared';

@Injectable({
    providedIn: 'root'
})
export class DefinitionsService {
    constructor(private transloco: TranslocoService) {}

    getDefinitions(eventNames: string[]) {
        const schema = clone(definitionsSchema) as Record<string, NcFormProperty>;

        replaceByProperties(
            schema,
            [
                '"base.action".properties.*.ui.label',
                '"events.aggregation".allOf.*.properties.*.rules.maxItems.errMsg',
                '"events.complex".properties.*.items.properties.*.rules.*.errMsg',
                '"events.complex".properties.*.items.properties.*.rules.customRule.*.errMsg',
                '"events.complex".properties.*.items.properties.*.ui.label',
                '"events.complex".properties.*.items.rules.*.errMsg',
                '"events.complex".properties.*.rules.*.errMsg',
                '"events.complex".properties.*.rules.customRule.*.errMsg',
                '"events.complex".properties.*.ui.label',
                '"events.complex".properties.*.ui.legend',
                '"events.correlation".allOf.*.properties.*.rules.maxItems.errMsg',
                '"types.aggregation".ui.label',
                '"types.atomic".ui.label',
                '"types.correlation".ui.label',
                'actions.ui.label',
                'actions.ui.widgetConfig.enumSource.*.label',
                'fields.ui.label'
            ],
            (key: string) => {
                if (/^[A-z\d]+\.[A-z\d]+\.[A-z\d]+\.[A-z\d]+$/.test(key)) {
                    const scope = key.split('.', 1)[0];

                    return this.transloco.translate(`${scope.toLowerCase()}.${key}`);
                } else {
                    return key;
                }
            }
        );

        schema['events.ids'].enum = eventNames;
        schema['events.ids'].ui.widgetConfig.enumSource = eventNames.map((name) => ({ value: name }));

        return schema;
    }
}
