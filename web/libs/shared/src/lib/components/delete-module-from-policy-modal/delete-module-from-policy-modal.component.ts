import { Component, ContentChild, Input, OnDestroy, OnInit, TemplateRef } from '@angular/core';
import { Router } from '@angular/router';
import { TranslocoService } from '@ngneat/transloco';
import { McButton } from '@ptsecurity/mosaic/button';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { McModalRef, McModalService, ModalSize } from '@ptsecurity/mosaic/modal';
import { combineLatestWith, first, pairwise, Subscription } from 'rxjs';

import { ErrorResponse } from '@soldr/api';
import { Policy } from '@soldr/models';
import { EntityModule, LanguageService, ModalInfoService } from '@soldr/shared';
import { ModulesInstancesFacade } from '@soldr/store/modules-instances';

@Component({
    selector: 'soldr-delete-module-from-policy-modal',
    templateUrl: './delete-module-from-policy-modal.component.html',
    styleUrls: ['./delete-module-from-policy-modal.component.scss']
})
export class DeleteModuleFromPolicyModalComponent implements OnInit, OnDestroy {
    @Input() module: EntityModule;
    @Input() policy: Policy;

    @ContentChild(McButton) button: McButton;

    isDeletingModule$ = this.modulesInstancesFacade.isDeletingModule$;
    isDisablingModule$ = this.modulesInstancesFacade.isDisablingModule$;
    language$ = this.languageService.current$;
    modal: McModalRef;
    themePalette = ThemePalette;
    subscription = new Subscription();

    constructor(
        private languageService: LanguageService,
        private modalInfoService: ModalInfoService,
        private modalService: McModalService,
        private modulesInstancesFacade: ModulesInstancesFacade,
        private router: Router,
        private transloco: TranslocoService
    ) {}

    ngOnInit(): void {
        const deletingSubscription = this.isDeletingModule$
            .pipe(pairwise(), combineLatestWith(this.modulesInstancesFacade.deleteError$))
            .subscribe(([[oldValue, newValue], deleteError]) => {
                if (oldValue && !newValue) {
                    this.afterCloseModal(
                        deleteError,
                        this.transloco.translate('shared.Shared.ModulesConfig.ErrorText.Delete')
                    );
                }
            });
        this.subscription.add(deletingSubscription);

        const disablingSubscription = this.isDisablingModule$
            .pipe(pairwise(), combineLatestWith(this.modulesInstancesFacade.disableError$))
            .subscribe(([[oldValue, newValue], disableError]) => {
                if (oldValue && !newValue && this.modal?.getInstance()) {
                    this.afterCloseModal(
                        disableError,
                        this.transloco.translate('shared.Shared.ModulesConfig.ErrorText.Disable')
                    );
                }
            });
        this.subscription.add(disablingSubscription);
    }

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
    }

    afterCloseModal(error: ErrorResponse, errorText: string) {
        this.modal.close();
        this.modal.afterClose.pipe(first()).subscribe(() => {
            if (error) {
                this.modalInfoService.openErrorInfoModal(errorText);
            } else {
                this.router.navigate(['/policies', this.policy.hash], { queryParams: { tab: 'modules' } });
            }
        });
    }

    open(content: TemplateRef<any>, footer: TemplateRef<any>) {
        if (!this.button.disabled) {
            this.modal = this.modalService.create({
                mcTitle: this.transloco.translate('policies.Policies.DeleteModule.ModalTitle.DeleteModuleFromPolicy'),
                mcContent: content,
                mcFooter: footer,
                mcSize: ModalSize.Small
            });

            this.modal.afterClose.subscribe(() => this.button?.focusViaKeyboard());
        }
    }

    delete() {
        this.modulesInstancesFacade.deleteModule(this.policy.hash, this.module.info.name);
    }

    disable() {
        this.modulesInstancesFacade.disableModule(this.policy.hash, this.module.info.name);
    }

    cancel() {
        this.modal.close();
    }
}
