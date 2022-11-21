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
import {
    AbstractControl,
    AsyncValidatorFn,
    FormControl,
    FormGroup,
    ValidationErrors,
    Validators
} from '@angular/forms';
import { TranslocoService } from '@ngneat/transloco';
import { McButton } from '@ptsecurity/mosaic/button';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { McModalRef, McModalService, ModalSize } from '@ptsecurity/mosaic/modal';
import { BehaviorSubject, combineLatestWith, first, map, Observable, pairwise, Subscription } from 'rxjs';

import { ErrorResponse } from '@soldr/api';
import { AgentsValidationService } from '@soldr/features/agents';
import { Agent } from '@soldr/models';
import { ENTITY_NAME_MAX_LENGTH, ModalInfoService } from '@soldr/shared';
import { TagDomain, TagsFacade } from '@soldr/store/tags';

@Component({
    selector: 'soldr-edit-agent-modal',
    templateUrl: './edit-agent-modal.component.html',
    styleUrls: ['./edit-agent-modal.component.scss']
})
export class EditAgentModalComponent implements OnInit, OnChanges, OnDestroy {
    @Input() agent: Agent;
    @Input() isUpdatingAgent: boolean;
    @Input() updateError: ErrorResponse;

    @Output() afterSave = new EventEmitter<void>();
    @Output() updateAgent = new EventEmitter<Agent>();

    @ContentChild(McButton) button: McButton;

    form: FormGroup<{ description: FormControl<string>; tags: FormControl<string[]> }>;
    themePalette = ThemePalette;
    modal: McModalRef;
    tags$ = this.tagsFacade.agentsTags$;
    subscription = new Subscription();

    private inputIsUpdatingAgent$ = new BehaviorSubject<boolean>(false);
    private inputUpdateError$ = new BehaviorSubject<ErrorResponse>(undefined);

    constructor(
        private agentsValidationService: AgentsValidationService,
        private modalInfoService: ModalInfoService,
        private modalService: McModalService,
        private tagsFacade: TagsFacade,
        private transloco: TranslocoService
    ) {}

    private defineForm() {
        this.form = new FormGroup({
            description: new FormControl(
                '',
                [Validators.required, Validators.maxLength(ENTITY_NAME_MAX_LENGTH)]
            ),
            tags: new FormControl([], [])
        });
    }

    private entityNameExistsValidator(exclude: string[] = []): AsyncValidatorFn {
        return (control: AbstractControl): Observable<ValidationErrors | null> =>
            this.agentsValidationService.getIsExistedAgentsByDescription(control.value as string, exclude).pipe(
                map((exists) => (exists ? { entityNameExists: true } : null)),
                first()
            );
    }

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
    }

    ngOnInit(): void {
        const isUpdatingAgentSubscription = this.inputIsUpdatingAgent$
            .pipe(pairwise(), combineLatestWith(this.inputUpdateError$))
            .subscribe(([[oldValue, newValue], updateError]) => {
                if (oldValue && !newValue) {
                    this.processAgentUpdate(
                        updateError,
                        this.transloco.translate('agents.Agents.EditAgent.ErrorText.UpdateAgent')
                    );
                }
            });
        this.subscription.add(isUpdatingAgentSubscription);
    }

    ngOnChanges({ isUpdatingAgent, updateError }: SimpleChanges) {
        if (isUpdatingAgent) {
            this.inputIsUpdatingAgent$.next(isUpdatingAgent.currentValue as boolean);
        }

        if (updateError) {
            this.inputUpdateError$.next(updateError.currentValue as ErrorResponse);
        }
    }

    processAgentUpdate(error: ErrorResponse, text?: string) {
        if (!error) {
            this.afterSave.emit();
        }
        this.modal?.close();
        this.modal?.afterClose.pipe(first()).subscribe(() => {
            if (error) {
                this.modalInfoService.openErrorInfoModal(text);
            }
        });
    }

    open(content: TemplateRef<any>, footer: TemplateRef<any>) {
        if (!this.button.disabled) {
            this.tagsFacade.fetchTags(TagDomain.Agents);

            this.defineForm();

            this.form.patchValue({
                description: this.agent.description,
                tags: [...(this.agent.info.tags || [])]
            });

            this.modal = this.modalService.create({
                mcTitle: this.transloco.translate('agents.Agents.EditAgent.ModalTitle.Agent'),
                mcContent: content,
                mcFooter: footer,
                mcSize: ModalSize.Small
            });
        }
    }

    save() {
        setTimeout(() => {
            if (!this.form.valid) {
                return;
            }

            this.updateAgent.emit({
                ...this.agent,
                description: this.form.get('description').value,
                info: {
                    ...this.agent.info,
                    tags: this.form.get('tags').value
                }
            });
        });
    }

    cancel() {
        this.modal.close();
    }
}
