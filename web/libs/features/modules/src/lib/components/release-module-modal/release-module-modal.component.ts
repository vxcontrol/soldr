import { Component, ContentChild, TemplateRef } from '@angular/core';
import { FormBuilder, Validators } from '@angular/forms';
import { TranslocoService } from '@ngneat/transloco';
import { McButton } from '@ptsecurity/mosaic/button';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { McModalRef, McModalService } from '@ptsecurity/mosaic/modal';
import { pairwise, shareReplay, Subscription, take, withLatestFrom } from 'rxjs';

import { ModelsModuleS } from '@soldr/api';
import { getChangelogVersionRecordFormModel } from '@soldr/features/modules';
import { LANGUAGES } from '@soldr/i18n';
import { LanguageService, ModuleVersionPipe } from '@soldr/shared';
import { ModuleEditFacade } from '@soldr/store/modules';

import { ChangelogVersionRecord } from '../../types';
import { getChangelogVersionModel } from '../../utils/get-changelog-version-model';

@Component({
    selector: 'soldr-release-module-modal',
    templateUrl: './release-module-modal.component.html',
    styleUrls: ['./release-module-modal.component.scss']
})
export class ReleaseModuleModalComponent {
    @ContentChild(McButton) button: McButton;

    error$ = this.moduleEditFacade.releaseError$;
    form = this.formBuilder.group({
        date: this.formBuilder.control(''),
        version: this.formBuilder.control({ value: '', disabled: true }),
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
    isReleasing$ = this.moduleEditFacade.isReleasingModule$;
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
        this.form.reset();
        this.modal.close();
    }

    release() {
        this.moduleEditFacade.isReleasingModule$
            .pipe(pairwise(), withLatestFrom(this.error$), take(2))
            .subscribe(([[oldValue, newValue], error]) => {
                if (oldValue && !newValue) {
                    if (!error) {
                        this.modal.close();
                    }
                }
            });
        this.moduleEditFacade.releaseModule(getChangelogVersionModel(this.form.value as ChangelogVersionRecord));
    }

    private initForm() {
        this.module$
            .pipe(shareReplay({ bufferSize: 1, refCount: false }), take(1))
            .subscribe((module: ModelsModuleS) => {
                const currentVersion = new ModuleVersionPipe().transform(module.info.version);
                const record = module.changelog[currentVersion];

                this.form.setValue(getChangelogVersionRecordFormModel(currentVersion, record));
            });
    }
}
