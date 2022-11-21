import { Component, EventEmitter, Input, OnChanges, Output, SimpleChanges } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';

import { clone, EntityModule, LanguageService, NcformSchema, PropertyType, replaceByProperties } from '@soldr/shared';

import { NcformWrapperApi } from '../ncform-wrapper/ncform-wrapper.component';

@Component({
    selector: 'soldr-module-config',
    templateUrl: './module-config.component.html',
    styleUrls: ['./module-config.component.scss']
})
export class ModuleConfigComponent implements OnChanges {
    @Input() module: EntityModule;
    @Input() isReadOnly: boolean;

    @Output() changeConfig = new EventEmitter<any>();

    config: any;
    schema: NcformSchema;

    private api: NcformWrapperApi;

    constructor(private languageService: LanguageService, private transloco: TranslocoService) {}

    ngOnChanges({ module }: SimpleChanges): void {
        if (module?.currentValue) {
            this.schema = this.processSchema(clone(this.module.config_schema));
            this.config = this.module.current_config;
        }
    }

    onRegisterApi(api: NcformWrapperApi) {
        this.api = api;
    }

    processSchema(schema: any) {
        replaceByProperties(
            schema.properties,
            [
                '*.rules.*.errMsg',
                '*.rules.customRule.*.errMsg',
                '*.properties.*.rules.*.errMsg',
                '*.properties.*.rules.customRule.*.errMsg',
                '*.items.rules.*.errMsg',
                '*.items.rules.customRule.*.errMsg',
                '*.items.properties.*.rules.*.errMsg',
                '*.items.properties.*.rules.customRule.*.errMsg'
            ],
            (key: string) =>
                /^[A-z\d]+\.[A-z\d]+\.[A-z\d]+\.[A-z\d]+$/.test(key) ? this.transloco.translate(`common.${key}`) : key
        );

        this.localizeProperties(
            schema.properties as Record<string, any>,
            this.module?.locale.config as Record<string, any>
        );

        return schema;
    }

    validate() {
        return this.api.validate();
    }

    getModel() {
        return this.api.getValue();
    }

    reset() {
        this.config = clone(this.module.current_config);
    }

    get isDirty() {
        return this.api?.getIsDirty();
    }

    private localizeProperties(properties: Record<string, any>, locales: Record<string, any>) {
        const lang = this.languageService.lang;

        Object.keys(properties)
            .filter((property) => !['priority'].includes(property))
            .forEach((property) => {
                const ui = (properties[property].ui || {}) as Record<string, any>;

                const propertyLocale = locales[property];

                ui.label = (propertyLocale[lang] || {}).title;

                if (
                    properties[property].type === PropertyType.ARRAY ||
                    properties[property].type === PropertyType.OBJECT
                ) {
                    ui.legend = (propertyLocale[lang] || {}).description;
                } else {
                    ui.description = (propertyLocale[lang] || {}).description;
                }
            });
    }
}
