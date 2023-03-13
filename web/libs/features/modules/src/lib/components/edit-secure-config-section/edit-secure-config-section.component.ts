import { Component, ElementRef, Input, OnInit, ViewChild } from '@angular/core';
import { FormBuilder, FormGroup, Validators } from '@angular/forms';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { first, pairwise, reduce, startWith, Subject, Subscription, take, withLatestFrom } from 'rxjs';

import {
    clone,
    getChangesArrays,
    getEmptySchema,
    localizeSchemaAdditionalKeys,
    NcformSchema,
    NcformWrapperApi,
    usedPropertyTypes
} from '@soldr/shared';
import { ModuleEditFacade } from '@soldr/store/modules';

import { DialogsService } from '../../services';
import { ConfigurationItem, ModuleSection, SecureConfigurationItem } from '../../types';
import { getSecureConfigModelFromSchema, getSecureConfigSchemaFromModel, applyDiff } from '../../utils';
import { unwrapFormItems } from '../../utils/unwrap-form-items';
import {
    correctDefaultValueValidator,
    formItemFieldsValidator,
    formItemNameValidator,
    overlappingNamesValidator
} from '../../validators';

@Component({
    selector: 'soldr-edit-secure-config-section',
    templateUrl: './edit-secure-config-section.component.html',
    styleUrls: ['./edit-secure-config-section.component.scss']
})
export class EditSecureConfigSectionComponent implements OnInit, ModuleSection {
    @Input() readOnly: boolean;

    @ViewChild('formElement') formElement: ElementRef<HTMLFormElement>;

    activeTabIndex = 0;
    defaultConfigFormApi: NcformWrapperApi;
    defaultSchema: NcformSchema;
    defaultModel: any;
    form = this.formBuilder.group({
        params: this.formBuilder.array<SecureConfigurationItem>([])
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

    ngOnInit(): void {
        const initFormSubscription = this.moduleEditFacade.secureConfigSchemaModel$
            .pipe(startWith(undefined), pairwise())
            .subscribe(([oldSchema, schema]) => {
                const prevSchema: NcformSchema = oldSchema || getEmptySchema();
                const oldModel = getSecureConfigModelFromSchema(prevSchema, true);
                const model = getSecureConfigModelFromSchema(schema, true);
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
            .valueChanges.subscribe((params: SecureConfigurationItem[]) => {
                const names = params.map(({ name }) => name);
                const hasOverlappedNames = new Set(names).size < names.length;

                if (!hasOverlappedNames) {
                    this.moduleEditFacade.updateSecureConfigSchema(getSecureConfigSchemaFromModel(params, false));
                }
            });
        this.subscription.add(updateSchemaSubscription);

        const defaultSubscription = this.moduleEditFacade.module$
            .pipe(withLatestFrom(this.moduleEditFacade.changedSecureParams$))
            .subscribe(([module, changes]) => {
                const defaultSchema = unwrapFormItems(clone(module.secure_config_schema) as NcformSchema, changes);
                this.defaultSchema = localizeSchemaAdditionalKeys(
                    defaultSchema,
                    module.locale.secure_config_additional_args
                );
                this.defaultModel = module.secure_default_config;
            });
        this.subscription.add(defaultSubscription);
    }

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
    }

    get params() {
        return this.form.controls.params;
    }

    onChangeDefaultConfig(model: Record<string, any>) {
        this.moduleEditFacade.updateSecureDefaultConfig(model);
    }

    addParamToConfig() {
        this.moduleEditFacade.addSecureConfigParam();
    }

    removeParamFromConfig(name: string) {
        this.dialogs.showRemoveDialog().subscribe((confirmed) => {
            if (confirmed) {
                this.moduleEditFacade.removeSecureConfigParam(name);
            }
        });
    }

    removeAllSecureParamsFromConfig() {
        this.dialogs.showRemoveDialog(true).subscribe((confirmed) => {
            if (confirmed) {
                this.moduleEditFacade.removeAllSecureConfigParams();
            }
        });
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
            this.moduleEditFacade.setValidationState('secureConfiguration', status);
        });

        return result$;
    }

    private getParamFormGroup(param: SecureConfigurationItem) {
        return this.formBuilder.group(
            {
                required: [{ value: param.required, disabled: this.readOnly }],
                serverOnly: [{ value: param.serverOnly, disabled: this.readOnly }],
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

    onRegisterApiDefaultConfigForm(api: NcformWrapperApi) {
        this.defaultConfigFormApi = api;
    }
}
