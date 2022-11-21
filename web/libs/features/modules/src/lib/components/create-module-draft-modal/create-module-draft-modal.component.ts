import { Component, ContentChild, TemplateRef } from '@angular/core';
import { FormBuilder, Validators } from '@angular/forms';
import { TranslocoService } from '@ngneat/transloco';
import { McButton } from '@ptsecurity/mosaic/button';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { McModalRef, McModalService } from '@ptsecurity/mosaic/modal';
import { combineLatestWith, pairwise, shareReplay, Subscription, take, withLatestFrom } from 'rxjs';
import * as semver from 'semver';

import { getChangelogVersionRecordFormModel } from '@soldr/features/modules';
import { LANGUAGES } from '@soldr/i18n';
import { LanguageService, ModuleVersionPipe } from '@soldr/shared';
import { ModuleEditFacade } from '@soldr/store/modules';

import { ChangelogVersionRecord } from '../../types';
import { getChangelogVersionModel } from '../../utils';
import { draftVersion } from '../../validators';

@Component({
    selector: 'soldr-create-module-draft-modal',
    templateUrl: './create-module-draft-modal.component.html',
    styleUrls: ['./create-module-draft-modal.component.scss']
})
export class CreateModuleDraftModalComponent {
    @ContentChild(McButton) button: McButton;

    error$ = this.moduleEditFacade.releaseError$;
    form = this.formBuilder.group({
        date: this.formBuilder.control(''),
        version: this.formBuilder.control(''),
        locales: this.formBuilder.group({
            [LANGUAGES.ru]: this.formBuilder.group({
                title: this.formBuilder.control('', Validators.required),
                description: this.formBuilder.control('', Validators.required)
            }),
            [LANGUAGES.en]: this.formBuilder.group({
                title: this.formBuilder.control('', Validators.required),
                description: this.formBuilder.control('', Validators.required)
            })
        })
    });
    isCreatingDraft$ = this.moduleEditFacade.isCreatingDraft$;
    lang = this.languageService.lang;
    modal: McModalRef;
    module$ = this.moduleEditFacade.module$;
    subscription = new Subscription();
    themePalette = ThemePalette;

    constructor(
        private moduleEditFacade: ModuleEditFacade,
        private languageService: LanguageService,
        private modalService: McModalService,
        private transloco: TranslocoService,
        private formBuilder: FormBuilder
    ) {}

    open(title: TemplateRef<any>, content: TemplateRef<any>, footer: TemplateRef<any>) {
        this.initForm();

        this.modal = this.modalService.create({
            mcTitle: title,
            mcContent: content,
            mcFooter: footer
        });

        this.modal.afterClose.pipe(take(1)).subscribe(() => {
            this.moduleEditFacade.resetOperationsError();
        });
    }

    cancel() {
        this.modal.close();
    }

    createDraft() {
        this.moduleEditFacade.isCreatingDraft$
            .pipe(pairwise(), withLatestFrom(this.error$), take(2))
            .subscribe(([[oldValue, newValue], error]) => {
                if (oldValue && !newValue) {
                    if (!error) {
                        this.modal.close();
                    }
                }
            });

        if (this.form.valid) {
            this.moduleEditFacade.createDraft(
                this.form.get('version').value,
                getChangelogVersionModel(this.form.value as ChangelogVersionRecord)
            );
        }
    }

    private initForm() {
        this.module$
            .pipe(
                shareReplay({ bufferSize: 1, refCount: false }),
                take(1),
                combineLatestWith(this.moduleEditFacade.moduleVersions$)
            )
            .subscribe(([module, versions]) => {
                const currentVersion = new ModuleVersionPipe().transform(module.info.version);
                const lastVersion = semver.rsort(
                    versions.map(({ info }) => new ModuleVersionPipe().transform(info.version))
                )[0];
                const record = module.changelog[currentVersion];
                const formModel = getChangelogVersionRecordFormModel(
                    new ModuleVersionPipe().transform(module.info.version),
                    record
                );

                formModel.locales.ru.description = this.transloco.translate(
                    'modules.Modules.ModuleEdit.Text.DefaultChangelogDescriptionRu'
                );
                formModel.locales.en.description = this.transloco.translate(
                    'modules.Modules.ModuleEdit.Text.DefaultChangelogDescriptionEn'
                );

                this.form.setValue(formModel);
                this.form.get('version').setValidators([draftVersion(lastVersion), Validators.required]);
            });
    }
}
