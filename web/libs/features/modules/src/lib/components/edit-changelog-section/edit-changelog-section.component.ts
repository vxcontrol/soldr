import { Component, ElementRef, Input, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { FormArray, FormBuilder, FormGroup, Validators } from '@angular/forms';
import { combineLatestWith, debounceTime, first, map, pairwise, startWith, Subject, Subscription, take } from 'rxjs';
import * as semver from 'semver';

import { ModelsChangelog, ModelsModuleSShort, ModuleState } from '@soldr/api';
import { applyDiff, getChangelogVersionRecordFormModel } from '@soldr/features/modules';
import { LANGUAGES } from '@soldr/i18n';
import { DEBOUNCING_DURATION, getChangesArrays, ModuleVersionPipe } from '@soldr/shared';
import { ModuleEditFacade } from '@soldr/store/modules';

import { ChangelogVersionRecord, ModuleSection } from '../../types';
import { getChangelogVersionModel } from '../../utils';

@Component({
    selector: 'soldr-edit-changelog-section',
    templateUrl: './edit-changelog-section.component.html',
    styleUrls: ['./edit-changelog-section.component.scss']
})
export class EditChangelogSectionComponent implements OnInit, OnDestroy, ModuleSection {
    @Input() readOnly: boolean;

    @ViewChild('formElement') formElement: ElementRef<HTMLFormElement>;

    currentVersion$ = this.moduleEditFacade.moduleVersions$.pipe(
        map(
            (versions) =>
                versions
                    .map((version) => new ModuleVersionPipe().transform(version.info.version))
                    .sort((b, a) => semver.compare(a, b))[0]
        )
    );
    form = this.formBuilder.group({ records: this.formBuilder.array([]) });

    private subscription = new Subscription();
    private validationState$ = new Subject<boolean>();

    constructor(private moduleEditFacade: ModuleEditFacade, private formBuilder: FormBuilder) {}

    get records() {
        return this.form.controls.records as FormArray;
    }

    ngOnInit(): void {
        this.watchStore();
        this.watchModel();
    }

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
    }

    deleteRecord(version: string) {
        this.moduleEditFacade.deleteChangelogRecord(version);
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
            this.moduleEditFacade.setValidationState('changelog', status);
        });

        return result$;
    }

    private watchStore() {
        const changelogSubscription = this.moduleEditFacade.changelog$
            .pipe(startWith(undefined), pairwise(), combineLatestWith(this.moduleEditFacade.moduleVersions$))
            .subscribe(([[oldChangelog, changelog], versions]) => {
                const oldModel = this.getFormModel(oldChangelog || {}).sort((b, a) =>
                    semver.compare(a.version, b.version)
                );
                const model = this.getFormModel(changelog).sort((b, a) => semver.compare(a.version, b.version));
                const oldKeys = Object.keys((oldChangelog || {}) as object);
                const keys = Object.keys(changelog as object);
                const diff = getChangesArrays(oldKeys, keys);
                diff[0] = diff[0].sort((b, a) => semver.compare(a, b));
                diff[1] = diff[1].sort((b, a) => semver.compare(a, b));

                applyDiff(
                    this.records,
                    diff,
                    oldModel,
                    model,
                    this.getRecordFormGroup.bind(this)(versions) as (param: ChangelogVersionRecord) => FormGroup,
                    'version'
                );
            });
        this.subscription.add(changelogSubscription);
    }

    private watchModel() {
        this.records.valueChanges
            .pipe(debounceTime(DEBOUNCING_DURATION))
            .subscribe((model: ChangelogVersionRecord[]) => {
                this.moduleEditFacade.updateChangelog(
                    this.getModel(this.records.getRawValue() as ChangelogVersionRecord[])
                );
            });
    }

    private getFormModel(model: ModelsChangelog): ChangelogVersionRecord[] {
        const result: ChangelogVersionRecord[] = [];

        for (const version of Object.keys(model)) {
            const item = model[version];
            result.push(getChangelogVersionRecordFormModel(version, item));
        }

        return result;
    }

    private getModel(model: ChangelogVersionRecord[]): ModelsChangelog {
        const result = {} as ModelsChangelog;

        for (const record of model) {
            result[record.version] = getChangelogVersionModel(record);
        }

        return result;
    }

    private getRecordFormGroup(versions: ModelsModuleSShort[]) {
        return (model: ChangelogVersionRecord) => {
            const version = versions.find(
                (item) => new ModuleVersionPipe().transform(item.info.version) === model.version
            );
            const isRelease = version?.state === ModuleState.Release;

            return this.formBuilder.group({
                version: this.formBuilder.control(model.version),
                date: this.formBuilder.control({ value: model.date, disabled: this.readOnly || isRelease }, [
                    Validators.required
                ]),
                locales: this.formBuilder.group({
                    [LANGUAGES.ru]: this.getLocaleRecord(model.locales.ru, isRelease),
                    [LANGUAGES.en]: this.getLocaleRecord(model.locales.en, isRelease)
                })
            });
        };
    }

    private getLocaleRecord(localeModel: ChangelogVersionRecord['locales']['ru' | 'en'], isRelease: boolean) {
        return this.formBuilder.group({
            title: this.formBuilder.control({ value: localeModel.title, disabled: this.readOnly || isRelease }, [
                Validators.required
            ]),
            description: this.formBuilder.control(
                { value: localeModel.description, disabled: this.readOnly || isRelease },
                [Validators.required]
            )
        });
    }
}
