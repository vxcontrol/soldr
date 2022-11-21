import { Component, ElementRef, Input, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { FormBuilder, FormControl, FormGroup, Validators } from '@angular/forms';
import { first, map, Subject, Subscription, take } from 'rxjs';

import { ModelsLocale, ModelsLocaleDesc, ModelsModuleLocaleDesc } from '@soldr/api';
import { LANGUAGES } from '@soldr/i18n';
import { ModuleEditFacade } from '@soldr/store/modules';

import { ModuleSection } from '../../types';

@Component({
    selector: 'soldr-edit-localization-section',
    templateUrl: './edit-localization-section.component.html',
    styleUrls: ['./edit-localization-section.component.scss']
})
export class EditLocalizationSectionComponent implements OnInit, OnDestroy, ModuleSection {
    @Input() readOnly: boolean;

    @ViewChild('formElement') formElement: ElementRef<HTMLFormElement>;

    form = this.formBuilder.group<{ [key in keyof ModelsLocale]: FormGroup<{ [p: string]: FormControl<any> }> }>({
        module: this.formBuilder.group({}),
        config: this.formBuilder.group({}),
        secure_config: this.formBuilder.group({}),
        fields: this.formBuilder.group({}),
        actions: this.formBuilder.group({}),
        events: this.formBuilder.group({}),
        action_config: this.formBuilder.group({}),
        event_config: this.formBuilder.group({}),
        tags: this.formBuilder.group({})
    });
    configParams$ = this.moduleEditFacade.localizationModel$.pipe(map((v) => Object.keys(v?.config || {})));
    secureConfigParams$ = this.moduleEditFacade.localizationModel$.pipe(
        map((v) => Object.keys(v?.secure_config || {}))
    );
    events$ = this.moduleEditFacade.localizationModel$.pipe(map((v) => Object.keys(v?.events || {})));
    eventConfig$ = this.moduleEditFacade.localizationModel$.pipe(map((v) => Object.keys(v?.event_config || {})));
    eventConfigKeys$ = this.moduleEditFacade.localizationModel$.pipe(
        map((v) =>
            Object.keys(v?.event_config || {}).reduce(
                (acc, item) => ({ ...acc, [item]: Object.keys(v.event_config[item] as object) }),
                {}
            )
        )
    );
    actions$ = this.moduleEditFacade.localizationModel$.pipe(map((v) => Object.keys(v?.actions || {})));
    actionConfig$ = this.moduleEditFacade.localizationModel$.pipe(map((v) => Object.keys(v?.action_config || {})));
    actionConfigKeys$ = this.moduleEditFacade.localizationModel$.pipe(
        map((v) =>
            Object.keys(v?.action_config || {}).reduce(
                (acc, item) => ({ ...acc, [item]: Object.keys(v.action_config[item] as object) }),
                {}
            )
        )
    );
    fields$ = this.moduleEditFacade.localizationModel$.pipe(map((v) => Object.keys(v?.fields || {})));
    tags$ = this.moduleEditFacade.localizationModel$.pipe(map((v) => Object.keys(v?.tags || {})));
    languages = Object.values(LANGUAGES).sort();
    languagesEnum = LANGUAGES;

    private subscription = new Subscription();
    private validationState$ = new Subject<boolean>();

    constructor(private formBuilder: FormBuilder, private moduleEditFacade: ModuleEditFacade) {}

    ngOnInit(): void {
        const modelSubscription = this.moduleEditFacade.localizationModel$.subscribe((model: ModelsLocale) => {
            this.form.setControl('module', this.getLocalizationFormGroup(model.module), { emitEvent: false });
            this.form.setControl('config', this.getKeysLocalizationFormGroup(model.config), { emitEvent: false });
            this.form.setControl('secure_config', this.getKeysLocalizationFormGroup(model.secure_config || {}), {
                emitEvent: false
            });
            this.form.setControl('events', this.getKeysLocalizationFormGroup(model.events), { emitEvent: false });
            this.form.setControl('event_config', this.getKeysGroupLocalizationFormGroup(model.event_config), {
                emitEvent: false
            });
            this.form.setControl('actions', this.getKeysLocalizationFormGroup(model.actions), { emitEvent: false });
            this.form.setControl('action_config', this.getKeysGroupLocalizationFormGroup(model.action_config), {
                emitEvent: false
            });
            this.form.setControl('fields', this.getKeysLocalizationFormGroup(model.fields), { emitEvent: false });
            this.form.setControl('tags', this.getKeysLocalizationFormGroup(model.tags), { emitEvent: false });

            if (this.readOnly) {
                this.form.disable();
            }
        });
        this.subscription.add(modelSubscription);

        const updateModelSubscription = this.form.valueChanges.subscribe((model: Partial<ModelsLocale>) => {
            this.moduleEditFacade.updateLocalizationModel(model);
        });
        this.subscription.add(updateModelSubscription);
    }

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
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
        this.formElement.nativeElement.requestSubmit();

        const result$ = this.validationState$.pipe(take(1));

        result$.subscribe((status) => {
            this.moduleEditFacade.setValidationState('localization', status);
        });

        return result$;
    }

    private getKeysGroupLocalizationFormGroup(data: Record<any, Record<any, ModelsModuleLocaleDesc>>) {
        return this.formBuilder.group(
            Object.keys(data).reduce(
                (acc, key) => ({ ...acc, [key]: this.getKeysLocalizationFormGroup(data[key]) }),
                {}
            ) as { [key: string]: any }
        );
    }

    private getKeysLocalizationFormGroup(data: Record<any, ModelsModuleLocaleDesc>) {
        return this.formBuilder.group(
            Object.keys(data).reduce(
                (acc, key) => ({ ...acc, [key]: this.getLocalizationFormGroup(data[key]) }),
                {}
            ) as { [key: string]: any }
        );
    }

    private getLocalizationFormGroup(data: ModelsModuleLocaleDesc) {
        return this.formBuilder.group(
            Object.values(LANGUAGES).reduce(
                (acc: any, lang: string) => ({
                    ...acc,
                    [lang]: this.getLocaleFormGroup(data[lang])
                }),
                {}
            ) as { [key: string]: any }
        );
    }

    private getLocaleFormGroup(data: ModelsLocaleDesc) {
        return this.formBuilder.group({
            title: [data.title, [Validators.required]],
            description: [data.description]
        });
    }
}
