import { Component, ElementRef, Input, OnDestroy, OnInit, ViewChild } from '@angular/core';
import {
    AbstractControl,
    AsyncValidatorFn,
    FormArray,
    FormBuilder,
    FormGroup,
    ValidationErrors,
    Validators
} from '@angular/forms';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import {
    filter,
    first,
    from,
    map,
    Observable,
    pairwise,
    reduce,
    skipWhile,
    startWith,
    Subject,
    Subscription,
    switchMap,
    take,
    toArray,
    withLatestFrom
} from 'rxjs';

import { ModelsModuleS, OptionsService } from '@soldr/api';
import { getModelFromSchema, getSchemaFromModel } from '@soldr/features/modules';
import {
    clone,
    getChangesArrays,
    getEmptySchema,
    localizeSchemaAdditionalKeys,
    NcFormProperty,
    NcFormReference,
    NcformSchema,
    NcformWrapperApi,
    PropertyType
} from '@soldr/shared';
import { ModuleEditFacade } from '@soldr/store/modules';
import { SharedFacade } from '@soldr/store/shared';

import { DefinitionsService, DialogsService } from '../../services';
import { ActionConfigurationItem, ConfigurationItem, ModuleSection } from '../../types';
import { applyDiff } from '../../utils';
import { unwrapFormItems } from '../../utils/unwrap-form-items';
import {
    correctDefaultValueValidator,
    formItemFieldsValidator,
    formItemNameValidator,
    overlappingNamesValidator
} from '../../validators';

const MAX_LENGTH_ACTION_NAME = 100;

@Component({
    selector: 'soldr-edit-actions-section',
    templateUrl: './edit-actions-section.component.html',
    styleUrls: ['./edit-actions-section.component.scss']
})
export class EditActionsSectionComponent implements OnInit, OnDestroy, ModuleSection {
    @Input() module: ModelsModuleS;
    @Input() readOnly: boolean;

    @ViewChild('formElement') formElement: ElementRef<HTMLFormElement>;

    activeKeyTabIndex = 0;
    defaultConfigFormApi: NcformWrapperApi;
    defaultSchema: NcformSchema;
    defaultModel: any;
    fields: string[] = [];
    form = this.formBuilder.group({
        actions: this.formBuilder.array<ActionConfigurationItem>([])
    });
    themePalette = ThemePalette;

    private subscription = new Subscription();
    private validationState$ = new Subject<boolean>();

    constructor(
        private moduleEditFacade: ModuleEditFacade,
        private optionsService: OptionsService,
        private formBuilder: FormBuilder,
        private dialogs: DialogsService,
        private definitions: DefinitionsService,
        private sharedFacade: SharedFacade
    ) {
        this.sharedFacade.fetchActions();
    }

    get actions() {
        return this.form.controls.actions as FormArray;
    }

    ngOnInit(): void {
        const initFormSubscription = this.moduleEditFacade.actionsConfigSchemaModel$
            .pipe(startWith(undefined), pairwise(), withLatestFrom(this.moduleEditFacade.changedActions$))
            .subscribe(([[oldSchema, schema], changes]) => {
                const prevSchema: NcformSchema = oldSchema || getEmptySchema();
                const oldModel = this.getActionsModelFromSchema(prevSchema);
                const model = this.getActionsModelFromSchema(schema);
                const namesOld = Object.keys(prevSchema.properties as object);
                const names = Object.keys(schema.properties as object);
                const diff = getChangesArrays(namesOld, names);

                const defaultSchema = unwrapFormItems(
                    {
                        ...clone(schema),
                        definitions: this.definitions.getDefinitions(names)
                    } as NcformSchema,
                    changes
                );
                this.defaultSchema = localizeSchemaAdditionalKeys(
                    defaultSchema,
                    this.module?.locale.actions_additional_args
                );

                applyDiff(
                    this.form.controls.actions,
                    diff,
                    oldModel,
                    model,
                    this.getActionsFormGroup.bind(this) as (param: ActionConfigurationItem) => FormGroup
                );

                if (diff[0]?.length === 0 && diff[1]?.length === 0) {
                    for (const [i, action] of model.entries()) {
                        const oldActionKeysModel = oldModel.find((item) => item.name === action.name);

                        if (!oldActionKeysModel) {
                            continue;
                        }
                        const oldKeysNames = oldActionKeysModel?.keys.map(({ name }) => name);
                        const keysNames = action?.keys.map(({ name }) => name);
                        const keysDiff = getChangesArrays(oldKeysNames, keysNames);
                        const formArray = this.actions.at(i).get('keys') as FormArray;
                        const fields = action?.fields;

                        this.actions.at(i).get('fields').setValue(fields, { emitEvent: false });

                        applyDiff(
                            formArray,
                            keysDiff,
                            oldActionKeysModel.keys,
                            action.keys,
                            this.getActionKeyFormGroup.bind(this) as (param: ConfigurationItem) => FormGroup
                        );

                        if (keysDiff[0].length > 0) {
                            this.activeKeyTabIndex = keysNames.length - 1;
                        }
                    }
                }
            });
        this.subscription.add(initFormSubscription);

        const updateSchemaSubscription = this.form
            .get('actions')
            .valueChanges.subscribe((actions: ActionConfigurationItem[]) => {
                const names = actions.map(({ name }) => name);
                const hasOverlappedNames = new Set(names).size < names.length;
                const hasOverlappedKeyNames = actions.some((action) => {
                    const keyNames = action.keys.map(({ name }) => name);

                    return new Set(keyNames).size < keyNames.length;
                });

                if (!hasOverlappedNames && !hasOverlappedKeyNames) {
                    this.moduleEditFacade.updateActionsSchema(this.getActionsSchemaFromModel(actions));
                }
            });
        this.subscription.add(updateSchemaSubscription);

        const defaultSubscription = this.moduleEditFacade.module$.subscribe((module) => {
            this.defaultModel = module.default_action_config;
            this.fields = Object.keys(module.fields_schema?.properties || {});
        });
        this.subscription.add(defaultSubscription);
    }

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
    }

    onChangeDefaultConfig(model: Record<string, any>) {
        this.moduleEditFacade.updateActionsDefaultConfig(model);
    }

    addEvent() {
        this.moduleEditFacade.addAction();
    }

    removeEvent(name: string) {
        this.dialogs.showRemoveDialog().subscribe((confirmed) => {
            if (confirmed) {
                this.moduleEditFacade.removeAction(name);
            }
        });
    }

    removeAllEvents() {
        this.dialogs.showRemoveDialog(true).subscribe((confirmed) => {
            if (confirmed) {
                this.moduleEditFacade.removeAllActions();
            }
        });
    }

    addKey(eventName: string) {
        this.moduleEditFacade.addActionKey(eventName);
    }

    deleteKey(eventName: string, keyName: string) {
        this.moduleEditFacade.removeActionKey(eventName, keyName);
    }

    onRegisterApiDefaultConfigForm(api: NcformWrapperApi) {
        this.defaultConfigFormApi = api;
    }

    onSubmitForm() {
        this.form.statusChanges
            .pipe(
                skipWhile((schemaStatus) => schemaStatus === 'PENDING'),
                first()
            )
            .subscribe((schemaStatus) => {
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
            this.moduleEditFacade.setValidationState('actions', status);
        });

        return result$;
    }

    private getActionKeyFormGroup(param: ConfigurationItem) {
        return this.formBuilder.group(
            {
                required: [{ value: param.required, disabled: this.readOnly }],
                name: [
                    { value: param.name, disabled: this.readOnly },
                    [Validators.required, formItemNameValidator(), overlappingNamesValidator()]
                ],
                type: [{ value: param.type, disabled: this.readOnly }, Validators.required],
                fields: [
                    { value: param.fields, disabled: this.readOnly },
                    [Validators.required, formItemFieldsValidator()]
                ]
            },
            { validators: [correctDefaultValueValidator()] }
        );
    }

    private getActionsFormGroup(param: ActionConfigurationItem) {
        return this.formBuilder.group(
            {
                name: [
                    { value: param.name, disabled: this.readOnly },
                    [
                        Validators.required,
                        Validators.maxLength(MAX_LENGTH_ACTION_NAME),
                        formItemNameValidator(),
                        overlappingNamesValidator()
                    ],
                    [this.actionNameExistsValidator()]
                ],
                priority: [{ value: param.priority, disabled: this.readOnly }, [Validators.required]],
                fields: [{ value: param.fields, disabled: this.readOnly }],
                config_fields: [{ value: param.config_fields, disabled: this.readOnly }, [formItemFieldsValidator()]],
                keys: this.formBuilder.array(param.keys.map((item) => this.getActionKeyFormGroup(item)))
            },
            {
                validators: []
            }
        );
    }

    private actionNameExistsValidator(): AsyncValidatorFn {
        return (control: AbstractControl): Promise<ValidationErrors | null> | Observable<ValidationErrors | null> =>
            this.sharedFacade.optionsActions$.pipe(
                first(),
                switchMap((v) => from(v)),
                withLatestFrom(this.moduleEditFacade.module$),
                filter(([action, module]) => action.module_name !== module.info.name && action.name === control.value),
                toArray(),
                map((found) => (found.length > 0 ? { actionNameExists: true } : null))
            );
    }

    private getActionsModelFromSchema(schema: NcformSchema): ActionConfigurationItem[] {
        const model: ActionConfigurationItem[] = [];

        if (typeof schema !== 'object' || Array.isArray(schema)) {
            return model;
        }
        if (typeof schema.properties !== 'object' || Array.isArray(schema.properties)) {
            return model;
        }

        for (const name of Object.keys(schema.properties)) {
            const actionType = schema.properties[name].allOf[0];
            const actionProperties = schema.properties[name].allOf[1];

            model.push({
                name,
                type: JSON.stringify(actionType),
                config_fields: JSON.stringify(
                    Object.keys(actionProperties)
                        .filter((k) => !['type', 'properties', 'required'].includes(k))
                        .reduce((acc, k) => {
                            const key = k as keyof Partial<NcFormProperty>;

                            return { ...acc, [key]: actionProperties[key] };
                        }, {} as Partial<NcFormProperty>),
                    undefined,
                    2
                ),
                keys: getModelFromSchema(
                    {
                        ...actionProperties,
                        ...{
                            properties: {
                                ...Object.keys(actionProperties.properties || {})
                                    .filter((k) => !['type', 'fields', 'priority'].includes(k))
                                    .reduce(
                                        (res, k) => ({ ...res, [k]: actionProperties.properties[k] }),
                                        {} as Record<string, any>
                                    )
                            }
                        }
                    },
                    true
                ),
                fields: actionProperties.properties?.fields?.default || [],
                priority: actionProperties.properties?.priority?.default || 1
            });
        }

        return model;
    }

    private getActionsSchemaFromModel(params: ActionConfigurationItem[]): NcformSchema {
        const schema: NcformSchema = {
            type: PropertyType.OBJECT,
            properties: {},
            required: [],
            additionalProperties: false
        };

        params.forEach((item) => {
            const name = item.name;

            schema.required.push(name);
            schema.properties[name] = {
                allOf: [
                    { $ref: '#/definitions/base.action' },
                    {
                        ...getSchemaFromModel(item.keys),
                        ...JSON.parse(item.config_fields || '{}')
                    }
                ] as Partial<NcFormReference>[]
            };

            const fields = item.fields;

            schema.properties[name].allOf[1].properties.fields = {
                ...(schema.properties[name].allOf[1].properties.fields || {}),
                ...{
                    type: PropertyType.ARRAY,
                    default: fields,
                    ...(fields && fields.length > 0
                        ? {
                              minItems: fields.length,
                              maxItems: fields.length
                          }
                        : {}),
                    items: {
                        type: PropertyType.STRING,
                        ...(fields && fields.length > 0 ? { enum: fields } : {})
                    }
                }
            };
            schema.properties[name].allOf[1].properties.priority = {
                ...(schema.properties[name].allOf[1].properties.priority || {}),
                ...{
                    default: item.priority || 1,
                    maximum: item.priority || 1,
                    minimum: item.priority || 1,
                    type: PropertyType.INTEGER
                }
            };
            schema.properties[name].allOf[1].required.push('fields', 'priority');
        });

        return schema;
    }
}
