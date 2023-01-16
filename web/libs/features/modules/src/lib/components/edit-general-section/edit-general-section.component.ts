import { Component, ElementRef, Input, OnChanges, OnDestroy, OnInit, SimpleChanges, ViewChild } from '@angular/core';
import { FormBuilder } from '@angular/forms';
import { TranslocoService } from '@ngneat/transloco';
import { filter, first, Observable, Subject, Subscription, take } from 'rxjs';

import { ModelsModuleInfo, ModelsModuleInfoOS } from '@soldr/api';
import { moduleOsListGroupByOs, ModuleVersionPipe } from '@soldr/shared';
import { ModuleEditFacade } from '@soldr/store/modules';
import { TagDomain, TagsFacade } from '@soldr/store/tags';

import { ModuleSection } from '../../types';
import { tagNameMaskForReplace, tagNameMask } from '../../utils/constants';

@Component({
    selector: 'soldr-edit-general-section',
    templateUrl: './edit-general-section.component.html',
    styleUrls: ['./edit-general-section.component.scss']
})
export class EditGeneralSectionComponent implements OnInit, OnChanges, OnDestroy, ModuleSection {
    @Input() readOnly: boolean;

    @ViewChild('formElement') formElement: ElementRef<HTMLFormElement>;

    allTags$: Observable<string[]> = this.tagsFacade.modulesTags$;
    form = this.formBuilder.group({
        name: this.formBuilder.control({ value: '', disabled: true }),
        template: this.formBuilder.control({ value: '', disabled: true }),
        version: this.formBuilder.control({ value: '', disabled: true }),
        os: this.formBuilder.control<string[]>([]),
        tags: this.formBuilder.control<string[]>([])
    });
    info: ModelsModuleInfo;
    moduleOsListGroupByOs = moduleOsListGroupByOs;
    tagNameMask = tagNameMask;
    tagNameMaskForReplace = tagNameMaskForReplace;

    private subscription = new Subscription();
    private validationState$ = new Subject<boolean>();

    constructor(
        private formBuilder: FormBuilder,
        private moduleEditFacade: ModuleEditFacade,
        private tagsFacade: TagsFacade,
        private transloco: TranslocoService
    ) {
        this.tagsFacade.fetchTags(TagDomain.Modules);
    }

    get template() {
        const template = this.info?.template;

        return template
            ? this.transloco.translate(
                  `modules.Modules.CreateModule.SelectItem.${template[0].toUpperCase()}${template.slice(1)}`
              )
            : '';
    }

    ngOnInit(): void {
        const moduleSubscription = this.moduleEditFacade.module$.pipe(filter(Boolean)).subscribe((module) => {
            this.info = module.info;

            this.initForm();
        });
        this.subscription.add(moduleSubscription);

        const updateModelSubscription = this.form.valueChanges.subscribe(() => {
            this.moduleEditFacade.updateGeneralSection(this.getModel());
        });
        this.subscription.add(updateModelSubscription);
    }

    ngOnChanges({ readOnly }: SimpleChanges): void {
        if (readOnly) {
            if (this.readOnly) {
                this.form.controls.os.disable({ emitEvent: false });
                this.form.controls.tags.disable({ emitEvent: false });
            } else {
                this.form.controls.os.enable({ emitEvent: false });
                this.form.controls.tags.enable({ emitEvent: false });
            }
        }
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
        this.formElement.nativeElement.dispatchEvent(new Event('submit'));

        const result$ = this.validationState$.pipe(take(1));

        result$.subscribe((status) => {
            this.moduleEditFacade.setValidationState('general', status);
        });

        return result$;
    }

    private initForm() {
        this.form.patchValue(
            {
                name: this.info.name,
                template: this.template,
                version: new ModuleVersionPipe().transform(this.info.version),
                os: this.toOsList(this.info.os),
                tags: this.info.tags
            },
            {
                emitEvent: false
            }
        );
    }

    private getModel() {
        return {
            ...this.info,
            os: this.getModuleOsInfoFromOsList(this.form.controls.os.value || []),
            tags: this.form.controls.tags.value
        };
    }

    private toOsList(osInfo: ModelsModuleInfoOS) {
        return Object.keys(osInfo).reduce((acc, family) => {
            const item = osInfo[family].map((arch) => `${family}:${arch}`);

            return [...acc, ...item];
        }, [] as string[]);
    }

    private getModuleOsInfoFromOsList(osList: string[]) {
        return osList.reduce((acc, item) => {
            const [family, arch] = item.split(':');

            if (!acc[family]) {
                acc[family] = [];
            }

            acc[family] = [...acc[family], arch];

            return acc;
        }, {} as Record<string, string[]>);
    }
}
