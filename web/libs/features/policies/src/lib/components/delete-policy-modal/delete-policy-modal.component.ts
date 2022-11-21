import { Component, ContentChild, EventEmitter, Input, OnDestroy, OnInit, Output, TemplateRef } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';
import { McButton } from '@ptsecurity/mosaic/button';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { McModalRef, McModalService, ModalSize } from '@ptsecurity/mosaic/modal';
import { combineLatestWith, first, pairwise, Subscription } from 'rxjs';

import { ErrorResponse } from '@soldr/api';
import { Policy } from '@soldr/models';
import { LanguageService, ModalInfoService } from '@soldr/shared';
import { PoliciesFacade } from '@soldr/store/policies';

@Component({
    selector: 'soldr-delete-policy-modal',
    templateUrl: './delete-policy-modal.component.html',
    styleUrls: ['./delete-policy-modal.component.scss']
})
export class DeletePolicyModalComponent implements OnInit, OnDestroy {
    @Input() policy: Policy;

    @Output() afterDelete = new EventEmitter();

    @ContentChild(McButton) button: McButton;

    isDeletingPolicy$ = this.policiesFacade.isDeletingPolicy$;
    language$ = this.languageService.current$;
    modal: McModalRef;
    themePalette = ThemePalette;
    subscription = new Subscription();

    constructor(
        private languageService: LanguageService,
        private modalInfoService: ModalInfoService,
        private modalService: McModalService,
        private policiesFacade: PoliciesFacade,
        private transloco: TranslocoService
    ) {}

    ngOnInit(): void {
        const deletingSubscription = this.isDeletingPolicy$
            .pipe(pairwise(), combineLatestWith(this.policiesFacade.deleteError$))
            .subscribe(([[oldValue, newValue], deleteError]) => {
                if (oldValue && !newValue) {
                    this.modal.close();
                    this.afterCloseModal(deleteError);
                }
            });
        this.subscription.add(deletingSubscription);
    }

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
    }

    afterCloseModal(error: ErrorResponse) {
        this.modal.afterClose.pipe(first()).subscribe(() => {
            if (error) {
                this.modalInfoService.openErrorInfoModal(
                    this.transloco.translate('policies.Policies.DeletePolicy.ErrorText.DeletePolicy')
                );
            } else {
                this.afterDelete.emit();
            }
        });
    }

    open(content: TemplateRef<any>, footer: TemplateRef<any>) {
        if (!this.button.disabled) {
            this.modal = this.modalService.create({
                mcTitle: this.transloco.translate('policies.Policies.DeletePolicy.ModalTitle.DeletePolicy'),
                mcContent: content,
                mcFooter: footer,
                mcSize: ModalSize.Small
            });

            this.modal.afterClose.subscribe(() => this.button?.focusViaKeyboard());
        }
    }

    delete() {
        this.policiesFacade.deletePolicy(this.policy.hash);
    }

    cancel() {
        this.modal.close();
    }
}
