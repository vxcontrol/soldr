import { Component, ElementRef, Input, OnDestroy, OnInit, ViewChild } from '@angular/core';
import {
    AbstractControl,
    AsyncValidatorFn,
    FormArray,
    FormBuilder,
    FormGroup,
    ValidationErrors,
    ValidatorFn,
    Validators
} from '@angular/forms';
import { TranslocoService } from '@ngneat/transloco';
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

import { OptionsService } from '@soldr/api';
import { getModelFromSchema, getSchemaFromModel } from '@soldr/features/modules';
import {
    clone,
    getChangesArrays,
    getEmptySchema,
    ListItem,
    NcFormProperty,
    NcFormReference,
    NcformSchema,
    NcformWrapperApi,
    PropertyType
} from '@soldr/shared';
import { ModuleEditFacade } from '@soldr/store/modules';
import { SharedFacade } from '@soldr/store/shared';

import { DefinitionsService, DialogsService } from '../../services';
import { ConfigurationItem, EventConfigurationItem, EventConfigurationItemType, ModuleSection } from '../../types';
import { applyDiff } from '../../utils';
import { unwrapFormItems } from '../../utils/unwrap-form-items';
import {
    correctDefaultValueValidator,
    formItemFieldsValidator,
    formItemNameValidator,
    overlappingNamesValidator
} from '../../validators';

const MAX_LENGTH_EVENT_NAME = 100;

@Component({
    selector: 'soldr-edit-events-section',
    templateUrl: './edit-events-section.component.html',
    styleUrls: ['./edit-events-section.component.scss']
})
export class EditEventsSectionComponent implements OnInit, ModuleSection, OnDestroy {
    @Input() readOnly: boolean;

    @ViewChild('formElement') formElement: ElementRef<HTMLFormElement>;

    activeKeyTabIndex = 0;
    defaultConfigFormApi: NcformWrapperApi;
    defaultSchema: NcformSchema;
    defaultModel: any;
    fields: string[] = [];
    form = this.formBuilder.group({
        events: this.formBuilder.array<EventConfigurationItem>([])
    });
    themePalette = ThemePalette;
    types = EventConfigurationItemType;
    typeList: ListItem[] = [
        { label: 'atomic', value: EventConfigurationItemType.Atomic },
        { label: 'aggregation', value: EventConfigurationItemType.Aggregation },
        { label: 'correlation', value: EventConfigurationItemType.Correlation }
    ];

    private subscription = new Subscription();
    private validationState$ = new Subject<boolean>();

    constructor(
        private moduleEditFacade: ModuleEditFacade,
        private optionsService: OptionsService,
        private formBuilder: FormBuilder,
        private transloco: TranslocoService,
        private dialogs: DialogsService,
        private definitions: DefinitionsService,
        private sharedFacade: SharedFacade
    ) {
        this.sharedFacade.fetchEvents();
    }

    get events(): FormArray {
        return this.form.controls.events;
    }

    ngOnInit(): void {
        const initFormSubscription = this.moduleEditFacade.eventsConfigSchemaModel$
            .pipe(startWith(undefined), pairwise(), withLatestFrom(this.moduleEditFacade.changedEvents$))
            .subscribe(([[oldSchema, schema], changes]) => {
                const prevSchema: NcformSchema = oldSchema || getEmptySchema();
                const oldModel = this.getEventsModelFromSchema(prevSchema);
                const model = this.getEventsModelFromSchema(schema);
                const namesOld = Object.keys(prevSchema.properties as object);
                const names = Object.keys(schema.properties as object);
                const diff = getChangesArrays(namesOld, names);

                this.defaultSchema = unwrapFormItems(
                    {
                        ...clone(schema),
                        definitions: this.definitions.getDefinitions(names)
                    } as NcformSchema,
                    changes
                );

                applyDiff(
                    this.form.controls.events,
                    diff,
                    oldModel,
                    model,
                    this.getEventsFormGroup.bind(this) as (param: EventConfigurationItem) => FormGroup
                );

                // changes inside properties or renamed
                if (diff[0]?.length === 0 && diff[1]?.length === 0) {
                    for (const [i, event] of model.entries()) {
                        const oldEventKeysModel = oldModel.find((item) => item.name === event.name);

                        if (!oldEventKeysModel) {
                            continue;
                        }
                        const oldKeysNames = oldEventKeysModel?.keys.map(({ name }) => name);
                        const keysNames = event?.keys.map(({ name }) => name);
                        const keysDiff = getChangesArrays(oldKeysNames, keysNames);
                        const formArray = this.events.at(i).get('keys') as FormArray;
                        const fields = event?.fields;

                        this.events.at(i).get('fields').setValue(fields, { emitEvent: false });

                        applyDiff(
                            formArray,
                            keysDiff,
                            oldEventKeysModel.keys,
                            event.keys,
                            this.getEventKeyFormGroup.bind(this) as (param: ConfigurationItem) => FormGroup
                        );

                        if (keysDiff[0].length > 0) {
                            this.activeKeyTabIndex = keysNames.length - 1;
                        }
                    }
                }
            });
        this.subscription.add(initFormSubscription);

        const updateSchemaSubscription = this.form
            .get('events')
            .valueChanges.subscribe((events: EventConfigurationItem[]) => {
                const names = events.map(({ name }) => name);
                const hasOverlappedNames = new Set(names).size < names.length;
                const hasOverlappedKeyNames = events.some((event) => {
                    const keyNames = event.keys.map(({ name }) => name);

                    return new Set(keyNames).size < keyNames.length;
                });

                if (!hasOverlappedNames && !hasOverlappedKeyNames) {
                    this.moduleEditFacade.updateEventsSchema(this.getEventsSchemaFromModel(events));
                }
            });
        this.subscription.add(updateSchemaSubscription);

        const defaultSubscription = this.moduleEditFacade.module$.subscribe((module) => {
            this.defaultModel = module.default_event_config;
            this.fields = Object.keys(module.fields_schema?.properties || {});
        });
        this.subscription.add(defaultSubscription);
    }

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
    }

    onChangeDefaultConfig(model: Record<string, any>) {
        this.moduleEditFacade.updateEventsDefaultConfig(model);
    }

    addEvent() {
        this.moduleEditFacade.addEvent();
    }

    removeEvent(name: string) {
        this.dialogs.showRemoveDialog().subscribe((confirmed) => {
            if (confirmed) {
                this.moduleEditFacade.removeEvent(name);
            }
        });
    }

    removeAllEvents() {
        this.dialogs.showRemoveDialog(true).subscribe((confirmed) => {
            if (confirmed) {
                this.moduleEditFacade.removeAllEvents();
            }
        });
    }

    addKey(eventName: string) {
        this.moduleEditFacade.addEventKey(eventName);
    }

    deleteKey(eventName: string, keyName: string) {
        this.moduleEditFacade.removeEventKey(eventName, keyName);
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
        this.formElement.nativeElement.requestSubmit();

        const result$ = this.validationState$.pipe(
            take(2),
            reduce((acc, value) => acc && value, true)
        );

        result$.subscribe((status) => {
            this.moduleEditFacade.setValidationState('events', status);
        });

        return result$;
    }

    private getEventKeyFormGroup(param: ConfigurationItem) {
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

    private getEventsFormGroup(param: EventConfigurationItem) {
        return this.formBuilder.group(
            {
                name: [
                    { value: param.name, disabled: this.readOnly },
                    [
                        Validators.required,
                        Validators.maxLength(MAX_LENGTH_EVENT_NAME),
                        formItemNameValidator(),
                        overlappingNamesValidator()
                    ],
                    [this.eventNameExistsValidator()]
                ],
                type: [{ value: param.type, disabled: this.readOnly }, [Validators.required]],
                fields: [{ value: param.fields, disabled: this.readOnly }],
                config_fields: [{ value: param.config_fields, disabled: this.readOnly }, [formItemFieldsValidator()]],
                keys: this.formBuilder.array(param.keys.map((item) => this.getEventKeyFormGroup(item)))
            },
            {
                validators: [this.eventConfigTypeEmptyObjectValidator(), this.eventConfigTypeEmptyListValidation()]
            }
        );
    }

    private eventNameExistsValidator(): AsyncValidatorFn {
        return (control: AbstractControl): Promise<ValidationErrors | null> | Observable<ValidationErrors | null> =>
            this.sharedFacade.optionsEvents$.pipe(
                first(),
                switchMap((v) => from(v)),
                withLatestFrom(this.moduleEditFacade.module$),
                filter(([event, module]) => event.module_name !== module.info.name && event.name === control.value),
                toArray(),
                map((found) => (found.length > 0 ? { eventNameExists: true } : null))
            );
    }

    private eventConfigTypeEmptyObjectValidator(): ValidatorFn {
        return (control: AbstractControl): ValidationErrors | null =>
            control.get('type').value !== EventConfigurationItemType.Atomic &&
            control.get('config_fields').value !== '{}'
                ? { eventConfigTypeEmptyObject: true }
                : null;
    }

    private eventConfigTypeEmptyListValidation(): ValidatorFn {
        return (control: AbstractControl): ValidationErrors | null =>
            control.get('type').value !== EventConfigurationItemType.Atomic && control.get('keys').value?.length > 0
                ? { eventConfigTypeEmptyList: true }
                : null;
    }

    private getEventsModelFromSchema(schema: NcformSchema | NcFormProperty): EventConfigurationItem[] {
        const model: EventConfigurationItem[] = [];

        if (typeof schema !== 'object' || Array.isArray(schema)) {
            return model;
        }
        if (typeof schema.properties !== 'object' || Array.isArray(schema.properties)) {
            return model;
        }

        for (const name of Object.keys(schema.properties)) {
            const eventType = schema.properties[name].allOf[0];
            const eventsProperty = schema.properties[name].allOf[1];

            model.push({
                name,
                type: JSON.stringify(eventType) as EventConfigurationItemType,
                config_fields: JSON.stringify(
                    Object.keys(eventsProperty)
                        .filter((k) => !['type', 'properties', 'required'].includes(k))
                        .reduce((acc, k) => {
                            const key = k as keyof Partial<NcFormProperty>;

                            return { ...acc, [key]: eventsProperty[key] };
                        }, {} as Partial<NcFormProperty>),
                    undefined,
                    2
                ),
                keys: getModelFromSchema(
                    {
                        ...eventsProperty,
                        ...{
                            properties: {
                                ...Object.keys(eventsProperty.properties || {})
                                    .filter((k) => !['type', 'actions', 'fields'].includes(k))
                                    .reduce(
                                        (res, k) => ({ ...res, [k]: eventsProperty.properties[k] }),
                                        {} as Record<string, any>
                                    )
                            }
                        }
                    },
                    true
                ),
                fields: (eventsProperty.properties?.fields && eventsProperty.properties.fields?.default) || []
            });
        }

        return model;
    }

    private getEventsSchemaFromModel(params: EventConfigurationItem[]): NcformSchema {
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
                    JSON.parse(item.type || '{}'),
                    {
                        ...getSchemaFromModel(item.keys),
                        ...JSON.parse(item.config_fields || '{}')
                    }
                ] as Partial<NcFormReference>[]
            };

            if (schema.properties[name].allOf[0].$ref === '#/definitions/events.atomic') {
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
                schema.properties[name].allOf[1].required.push('fields');
            }
        });

        return schema;
    }
}
