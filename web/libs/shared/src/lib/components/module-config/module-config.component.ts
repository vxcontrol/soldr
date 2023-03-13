import { ChangeDetectorRef, Component, EventEmitter, Input, OnChanges, Output, SimpleChanges } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';

import {
    clone,
    EntityModule,
    LanguageService,
    localizeSchemaAdditionalKeys,
    NcformSchema,
    PropertyType
} from '@soldr/shared';

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

    constructor(
        private languageService: LanguageService,
        private transloco: TranslocoService,
        private cdr: ChangeDetectorRef
    ) {}

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
        schema.properties = this.localizeProperties(
            schema.properties as Record<string, any>,
            this.module?.locale.config as Record<string, any>,
            this.module?.locale.config_additional_args
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

    onModelChange(model: any): void {
        this.cdr.detectChanges();
        this.changeConfig.emit(model);
    }

    get isDirty() {
        return this.api?.getIsDirty();
    }

    private localizeProperties(
        properties: Record<string, any>,
        locales: Record<string, any>,
        additional: Record<string, Record<string, string>>
    ) {
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

        return localizeSchemaAdditionalKeys(properties, additional);
    }
}
