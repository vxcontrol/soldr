import { Component, ElementRef, Input, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { FormArray, FormBuilder, FormGroup, Validators } from '@angular/forms';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { first, pairwise, reduce, startWith, Subject, Subscription, take } from 'rxjs';

import { getChangesArrays, getEmptySchema, NcformSchema, NcformWrapperApi, usedPropertyTypes } from '@soldr/shared';
import { ModuleEditFacade } from '@soldr/store/modules';

import { DialogsService } from '../../services';
import { ConfigurationItem, ModuleSection } from '../../types';
import { getModelFromSchema, getSchemaFromModel, applyDiff } from '../../utils';
import {
    correctDefaultValueValidator,
    formItemFieldsValidator,
    formItemNameValidator,
    overlappingNamesValidator
} from '../../validators';

@Component({
    selector: 'soldr-edit-config-section',
    templateUrl: './edit-config-section.component.html',
    styleUrls: ['./edit-config-section.component.scss']
})
export class EditConfigSectionComponent implements OnInit, OnDestroy, ModuleSection {
    @Input() readOnly: boolean;

    @ViewChild('formElement') formElement: ElementRef<HTMLFormElement>;

    activeTabIndex = 0;
    defaultConfigFormApi: NcformWrapperApi;
    defaultSchema: NcformSchema;
    defaultModel: any;
    form = this.formBuilder.group({
        params: this.formBuilder.array<ConfigurationItem>([])
    });
    highlightedTabIndex = -1;
    themePalette = ThemePalette;
    propertiesTypes = usedPropertyTypes;

    private subscription = new Subscription();
    private validationState$ = new Subject<boolean>();

    constructor(
        private moduleEditFacade: ModuleEditFacade,
        private formBuilder: FormBuilder,
        private dialogs: DialogsService
    ) {}

    get params(): FormArray {
        return this.form.controls.params;
    }

    ngOnInit(): void {
        const initFormSubscription = this.moduleEditFacade.configSchemaModel$
            .pipe(startWith(undefined), pairwise())
            .subscribe(([oldSchema, schema]) => {
                const prevSchema: NcformSchema = oldSchema || getEmptySchema();
                const oldModel = getModelFromSchema(prevSchema, true);
                const model = getModelFromSchema(schema, true);
                const namesOld = oldModel.map(({ name }) => name);
                const names = model.map(({ name }) => name);
                const diff = getChangesArrays(namesOld, names);
                const changed = names.filter((propName) => {
                    const prop = model.find(({ name }) => name === propName);
                    const propFromOldModel = oldModel.find(({ name }) => name === propName);

                    return prop && propFromOldModel ? prop.type !== propFromOldModel.type : false;
                });

                applyDiff(
                    this.form.controls.params,
                    [...diff, changed],
                    oldModel,
                    model,
                    this.getParamFormGroup.bind(this) as (param: ConfigurationItem) => FormGroup
                );

                if (diff[0].length > 0) {
                    this.activeTabIndex = names.length - 1;
                }
            });
        this.subscription.add(initFormSubscription);

        const updateSchemaSubscription = this.form
            .get('params')
            .valueChanges.subscribe((params: ConfigurationItem[]) => {
                const names = params.map(({ name }) => name);
                const hasOverlappedNames = new Set(names).size < names.length;

                if (!hasOverlappedNames) {
                    this.moduleEditFacade.updateConfigSchema(getSchemaFromModel(params, false, true));
                }
            });
        this.subscription.add(updateSchemaSubscription);

        const defaultSubscription = this.moduleEditFacade.module$.subscribe((module) => {
            this.defaultSchema = module.config_schema;
            this.defaultModel = module.default_config;
        });
        this.subscription.add(defaultSubscription);
    }

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
    }

    onChangeDefaultConfig(model: Record<string, any>) {
        this.moduleEditFacade.updateDefaultConfig(model);
    }

    addParamToConfig() {
        this.moduleEditFacade.addConfigParam();
    }

    removeParamFromConfig(name: string) {
        this.dialogs.showRemoveDialog().subscribe((confirmed) => {
            if (confirmed) {
                this.moduleEditFacade.removeConfigParam(name);
            }
        });
    }

    removeAllParamsFromConfig() {
        this.dialogs.showRemoveDialog(true).subscribe((confirmed) => {
            if (confirmed) {
                this.moduleEditFacade.removeAllConfigParams();
            }
        });
    }

    onRegisterApiDefaultConfigForm(api: NcformWrapperApi) {
        this.defaultConfigFormApi = api;
    }

    onSubmitForm() {
        this.form.statusChanges.pipe(first()).subscribe((schemaStatus) => {
            this.validationState$.next(schemaStatus === 'VALID');
        });
        setTimeout(() => {
            this.form.updateValueAndValidity();
        });

        this.defaultConfigFormApi?.validate().then((modelStatus) => {
            this.validationState$.next(modelStatus.result as boolean);
        });
    }

    validateForms() {
        this.formElement.nativeElement.dispatchEvent(new Event('submit'));

        const result$ = this.validationState$.pipe(
            take(2),
            reduce((acc, value) => acc && value, true)
        );

        result$.subscribe((status) => {
            this.moduleEditFacade.setValidationState('configuration', status);
        });

        return result$;
    }

    private getParamFormGroup(param: ConfigurationItem) {
        return this.formBuilder.group(
            {
                required: [{ value: param.required, disabled: this.readOnly }],
                name: [
                    { value: param.name, disabled: this.readOnly },
                    [Validators.required, formItemNameValidator(), overlappingNamesValidator()]
                ],
                type: [{ value: param.type, disabled: this.readOnly }, [Validators.required]],
                fields: [
                    { value: param.fields, disabled: this.readOnly },
                    [Validators.required, formItemFieldsValidator()]
                ]
            },
            {
                validators: [correctDefaultValueValidator()]
            }
        );
    }
}
