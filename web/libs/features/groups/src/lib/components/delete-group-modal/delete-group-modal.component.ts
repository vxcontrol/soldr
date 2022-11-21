import { Component, ContentChild, EventEmitter, Input, OnDestroy, OnInit, Output, TemplateRef } from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';
import { McButton } from '@ptsecurity/mosaic/button';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { McModalRef, McModalService, ModalSize } from '@ptsecurity/mosaic/modal';
import { combineLatestWith, first, Observable, pairwise, Subscription } from 'rxjs';

import { ErrorResponse } from '@soldr/api';
import { Group } from '@soldr/models';
import { LanguageService, ModalInfoService } from '@soldr/shared';
import { GroupsFacade } from '@soldr/store/groups';

@Component({
    selector: 'soldr-delete-group-modal',
    templateUrl: './delete-group-modal.component.html',
    styleUrls: ['./delete-group-modal.component.scss']
})
export class DeleteGroupModalComponent implements OnInit, OnDestroy {
    @Input() disabled: boolean;
    @Input() group: Group;

    @Output() afterDelete = new EventEmitter();

    @ContentChild('modalButton') button: McButton;

    isDeletingGroup$: Observable<boolean>;
    language$ = this.languageService.current$;
    modal: McModalRef;
    themePalette = ThemePalette;
    subscription = new Subscription();

    constructor(
        private groupsFacade: GroupsFacade,
        private languageService: LanguageService,
        private modalInfoService: ModalInfoService,
        private modalService: McModalService,
        private transloco: TranslocoService
    ) {}

    ngOnInit(): void {
        this.isDeletingGroup$ = this.groupsFacade.isDeletingGroup$;

        const deletingSubscription = this.isDeletingGroup$
            .pipe(pairwise(), combineLatestWith(this.groupsFacade.deleteError$))
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
                    this.transloco.translate('groups.Groups.EditGroup.ErrorText.DeleteGroup')
                );
            } else {
                this.afterDelete.emit();
            }
        });
    }

    open(content: TemplateRef<any>, footer: TemplateRef<any>) {
        if (!this.disabled) {
            this.modal = this.modalService.create({
                mcTitle: this.transloco.translate('groups.Groups.DeleteGroup.ModalTitle.DeleteGroup'),
                mcContent: content,
                mcFooter: footer,
                mcSize: ModalSize.Small
            });

            this.modal.afterClose.subscribe(() => this.button?.focusViaKeyboard());
        }
    }

    delete() {
        this.groupsFacade.deleteGroup(this.group.hash);
    }

    cancel() {
        this.modal.close();
    }
}
