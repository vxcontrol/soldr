import { Component, ElementRef, Input, OnInit, ViewChild } from '@angular/core';
import { FormBuilder, FormGroup, Validators } from '@angular/forms';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { first, pairwise, startWith, Subject, Subscription, take } from 'rxjs';

import { getModelFromSchema, getSchemaFromModel } from '@soldr/features/modules';
import { getChangesArrays, getEmptySchema, NcformSchema, usedPropertyTypes } from '@soldr/shared';
import { ModuleEditFacade } from '@soldr/store/modules';

import { DialogsService } from '../../services';
import { ConfigurationItem, ModuleSection } from '../../types';
import { applyDiff } from '../../utils';
import {
    correctDefaultValueValidator,
    formItemFieldsValidator,
    formItemNameValidator,
    overlappingNamesValidator
} from '../../validators';

@Component({
    selector: 'soldr-edit-fields-section',
    templateUrl: './edit-fields-section.component.html',
    styleUrls: ['./edit-fields-section.component.scss']
})
export class EditFieldsSectionComponent implements OnInit, ModuleSection {
    @Input() readOnly: boolean;

    @ViewChild('formElement') formElement: ElementRef<HTMLFormElement>;

    activeTabIndex = 0;
    form = this.formBuilder.group({
        fields: this.formBuilder.array<ConfigurationItem>([])
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
        const initFormSubscription = this.moduleEditFacade.fieldsSchemaModel$
            .pipe(startWith(undefined), pairwise())
            .subscribe(([oldSchema, schema]) => {
                const prevSchema: NcformSchema = oldSchema || getEmptySchema();
                const oldModel = getModelFromSchema(prevSchema, true);
                const model = getModelFromSchema(schema, true);
                const namesOld = Object.keys(prevSchema.properties as object);
                const names = Object.keys(schema.properties as object);
                const diff = getChangesArrays(namesOld, names);
                const changed = names.filter((propName) => {
                    const prop = model.find(({ name }) => name === propName);
                    const propFromOldModel = oldModel.find(({ name }) => name === propName);

                    return prop && propFromOldModel ? prop.type !== propFromOldModel.type : false;
                });

                applyDiff(
                    this.form.controls.fields,
                    [...diff, changed],
                    oldModel,
                    model,
                    this.getFieldFormGroup.bind(this) as (param: ConfigurationItem) => FormGroup
                );

                if (diff[0].length > 0) {
                    this.activeTabIndex = names.length - 1;
                }
            });
        this.subscription.add(initFormSubscription);

        const updateSchemaSubscription = this.form
            .get('fields')
            .valueChanges.subscribe((params: ConfigurationItem[]) => {
                const names = params.map(({ name }) => name);
                const hasOverlappedNames = new Set(names).size < names.length;

                if (!hasOverlappedNames) {
                    this.moduleEditFacade.updateFieldsSchema(getSchemaFromModel(params, true, true));
                }
            });
        this.subscription.add(updateSchemaSubscription);
    }

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
    }

    get fields() {
        return this.form.controls.fields;
    }

    addField() {
        this.moduleEditFacade.addField();
    }

    removeField(name: string) {
        this.dialogs.showRemoveDialog().subscribe((confirmed) => {
            if (confirmed) {
                this.moduleEditFacade.removeField(name);
            }
        });
    }

    removeAllFields() {
        this.dialogs.showRemoveDialog(true).subscribe((confirmed) => {
            if (confirmed) {
                this.moduleEditFacade.removeAllFields();
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
    }

    validateForms() {
        this.formElement.nativeElement.dispatchEvent(new Event('submit'));

        const result$ = this.validationState$.pipe(take(1));

        result$.subscribe((status) => {
            this.moduleEditFacade.setValidationState('fields', status);
        });

        return result$;
    }

    private getFieldFormGroup(param: ConfigurationItem) {
        return this.formBuilder.group(
            {
                required: [{ value: param.required, disabled: this.readOnly }],
                name: [
                    { value: param.name, disabled: this.readOnly },
                    [Validators.required, formItemNameValidator(true), overlappingNamesValidator()]
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
