import {
    Component,
    ContentChild,
    EventEmitter,
    Input,
    OnChanges,
    OnDestroy,
    OnInit,
    Output,
    SimpleChanges,
    TemplateRef
} from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';
import { McButton } from '@ptsecurity/mosaic/button';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { McModalRef, McModalService, ModalSize } from '@ptsecurity/mosaic/modal';
import { BehaviorSubject, combineLatestWith, first, pairwise, Subject, Subscription } from 'rxjs';

import { ErrorResponse } from '@soldr/api';
import { Agent } from '@soldr/models';
import { ModalInfoService } from '@soldr/shared';

@Component({
    selector: 'soldr-delete-agents-modal',
    templateUrl: './delete-agents-modal.component.html',
    styleUrls: ['./delete-agents-modal.component.scss']
})
export class DeleteAgentsModalComponent implements OnInit, OnChanges, OnDestroy {
    @Input() agents: Agent[];
    @Input() isDeletingAgent: boolean;
    @Input() deleteError: ErrorResponse;

    @Output() afterDelete = new EventEmitter();
    @Output() deleteAgents = new EventEmitter<number[]>();

    @ContentChild(McButton) button: McButton;

    modal: McModalRef;
    themePalette = ThemePalette;
    subscription = new Subscription();

    private inputIsDeletingAgent$ = new BehaviorSubject<boolean>(false);
    private inputDeleteError$ = new BehaviorSubject<ErrorResponse>(undefined);

    constructor(
        private modalInfoService: ModalInfoService,
        private modalService: McModalService,
        private transloco: TranslocoService
    ) {}

    ngOnInit(): void {
        const deletingSubscription = this.inputIsDeletingAgent$
            .pipe(pairwise(), combineLatestWith(this.inputDeleteError$))
            .subscribe(([[oldValue, newValue], deleteError]) => {
                if (oldValue && !newValue) {
                    this.modal.close();
                    this.afterCloseModal(deleteError);
                }
            });
        this.subscription.add(deletingSubscription);
    }

    ngOnChanges({ isDeletingAgent, deleteError }: SimpleChanges) {
        if (isDeletingAgent) {
            this.inputIsDeletingAgent$.next(isDeletingAgent.currentValue as boolean);
        }

        if (deleteError) {
            this.inputDeleteError$.next(deleteError.currentValue as ErrorResponse);
        }
    }

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
    }

    afterCloseModal(error: ErrorResponse) {
        this.modal.afterClose.pipe(first()).subscribe(() => {
            if (error) {
                this.modalInfoService.openErrorInfoModal(
                    this.transloco.translate('agents.Agents.DeleteAgent.ErrorText.DeleteAgent')
                );
            } else {
                this.afterDelete.emit();
            }
        });
    }

    open(content: TemplateRef<any>, footer: TemplateRef<any>) {
        if (!this.button.disabled) {
            this.modal = this.modalService.create({
                mcTitle: this.transloco.translate('agents.Agents.DeleteAgent.ModalTitle.DeleteAgents', {
                    count: this.agents.length
                }),
                mcContent: content,
                mcFooter: footer,
                mcSize: ModalSize.Small
            });

            this.modal.afterClose.subscribe(() => this.button?.focusViaKeyboard());
        }
    }

    delete() {
        this.deleteAgents.emit(this.agents.map(({ id }) => id));
    }

    cancel() {
        this.modal.close();
    }
}
