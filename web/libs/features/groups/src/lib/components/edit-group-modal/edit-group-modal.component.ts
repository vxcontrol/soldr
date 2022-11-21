import { Component, ContentChild, EventEmitter, Input, OnDestroy, OnInit, Output, TemplateRef } from '@angular/core';
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
import { Observable, Subscription, combineLatest, map, pairwise, filter, first, combineLatestWith } from 'rxjs';

import { ErrorResponse } from '@soldr/api';
import { Group } from '@soldr/models';
import { ENTITY_NAME_MAX_LENGTH, getCopyName, LanguageService, ModalInfoService } from '@soldr/shared';
import { GroupsFacade } from '@soldr/store/groups';
import { SharedFacade } from '@soldr/store/shared';
import { TagDomain, TagsFacade } from '@soldr/store/tags';

@Component({
    selector: 'soldr-edit-group-modal',
    templateUrl: './edit-group-modal.component.html',
    styleUrls: ['./edit-group-modal.component.scss']
})
export class EditGroupModalComponent implements OnInit, OnDestroy {
    @Input() group?: Group;
    @Input() isCopy?: boolean;
    @Input() redirect?: boolean;

    @Output() afterSave = new EventEmitter();

    @ContentChild(McButton) button: McButton;

    form: FormGroup<{ name: FormControl<string>; tags: FormControl<string[]> }>;
    isProcessingGroup$: Observable<boolean>;
    isLoading$: Observable<boolean>;
    language$ = this.languageService.current$;
    localizedGroupsNames: string[];
    modal: McModalRef;
    placeholder: string;
    subscription = new Subscription();
    tags$: Observable<string[]>;
    themePalette = ThemePalette;

    constructor(
        private groupsFacade: GroupsFacade,
        private languageService: LanguageService,
        private modalInfoService: ModalInfoService,
        private modalService: McModalService,
        private sharedFacade: SharedFacade,
        private tagsFacade: TagsFacade,
        private transloco: TranslocoService
    ) {}

    private loadDataToForm() {
        const localizedName = this.group?.info.name[this.languageService.lang] || '';
        const name = this.isCopy
            ? getCopyName(
                  localizedName,
                  this.localizedGroupsNames,
                  this.transloco.translate('common.Common.Pseudo.Text.CopyPostfix')
              )
            : localizedName;

        this.form.patchValue({
            name,
            tags: [...(this.group?.info.tags || [])]
        });
    }

    private defineForm() {
        this.form = new FormGroup({
            name: new FormControl<string>(
                '',
                [Validators.required, Validators.maxLength(ENTITY_NAME_MAX_LENGTH)]
            ),
            tags: new FormControl<string[]>([], [])
        });
    }

    private entityNameExistsValidator(exclude: string[] = []): AsyncValidatorFn {
        return (control: AbstractControl): Observable<ValidationErrors | null> =>
            this.groupsFacade
                .getIsExistedGroupsByName(control.value as string, exclude)
                .pipe(map((exists) => (exists ? { entityNameExists: true } : null)));
    }

    ngOnInit(): void {
        this.initRx();

        const creatingSubscription = this.groupsFacade.isCreatingGroup$
            .pipe(pairwise(), combineLatestWith(this.groupsFacade.createError$))
            .subscribe(([[oldValue, newValue], createError]) => {
                if (oldValue && !newValue) {
                    this.processGroupActions(
                        createError,
                        this.transloco.translate('groups.Groups.EditGroup.ErrorText.CreateGroup')
                    );
                }
            });
        this.subscription.add(creatingSubscription);

        const copyingSubscription = this.groupsFacade.isCopyingGroup$
            .pipe(pairwise(), combineLatestWith(this.groupsFacade.copyError$))
            .subscribe(([[oldValue, newValue], copyError]) => {
                if (oldValue && !newValue) {
                    this.processGroupActions(
                        copyError,
                        this.transloco.translate('groups.Groups.EditGroup.ErrorText.CopyGroup')
                    );
                }
            });
        this.subscription.add(copyingSubscription);

        const updatingSubscription = this.groupsFacade.isUpdatingGroup$
            .pipe(pairwise(), combineLatestWith(this.groupsFacade.updateError$))
            .subscribe(([[oldValue, newValue], updateError]) => {
                if (oldValue && !newValue) {
                    this.processGroupActions(
                        updateError,
                        this.transloco.translate('groups.Groups.EditGroup.ErrorText.UpdateGroup')
                    );
                }
            });
        this.subscription.add(updatingSubscription);
    }

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
    }

    processGroupActions(error: ErrorResponse, text?: string) {
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

    initRx() {
        this.tags$ = this.tagsFacade.groupsTags$;
        this.isLoading$ = combineLatest([
            this.sharedFacade.isLoadingAllGroups$,
            this.tagsFacade.isLoadingGroupsTags$
        ]).pipe(map(([isLoadingAllGroups, isLoadingGroupsTags]) => isLoadingAllGroups || isLoadingGroupsTags));

        const localizedGroupsNamesSubscription = this.sharedFacade.allGroups$
            .pipe(map((groups: Group[]) => groups.reduce((acc, { info }) => [...acc, info.name.ru, info.name.en], [])))
            .subscribe((v: string[]) => {
                this.localizedGroupsNames = v;
            });
        this.subscription.add(localizedGroupsNamesSubscription);
    }

    open(content: TemplateRef<any>, footer: TemplateRef<any>) {
        if (!this.button.disabled) {
            this.sharedFacade.fetchAllGroups();
            this.tagsFacade.fetchTags(TagDomain.Groups);

            this.defineForm();

            this.modal = this.modalService.create({
                mcTitle: this.group
                    ? this.isCopy
                        ? this.transloco.translate('groups.Groups.EditGroup.ModalTitle.CopyGroup')
                        : this.transloco.translate('groups.Groups.EditGroup.ModalTitle.EditGroup')
                    : this.transloco.translate('groups.Groups.EditGroup.ModalTitle.NewGroup'),
                mcContent: content,
                mcFooter: footer,
                mcSize: ModalSize.Small
            });

            const createModalSubscription = this.isLoading$
                .pipe(
                    pairwise(),
                    filter(([oldValue, newValue]) => oldValue && !newValue),
                    first()
                )
                .subscribe(() => {
                    this.loadDataToForm();
                });

            this.isProcessingGroup$ = this.group
                ? this.isCopy
                    ? this.groupsFacade.isCopyingGroup$
                    : this.groupsFacade.isUpdatingGroup$
                : this.groupsFacade.isCreatingGroup$;

            const afterCloseSubscription = this.modal.afterClose.subscribe(() => {
                createModalSubscription.unsubscribe();
            });
            this.subscription.add(afterCloseSubscription);
        }
    }

    save() {
        setTimeout(() => {
            if (!this.form.valid) {
                return;
            }

            if (!this.group) {
                this.groupsFacade.createGroup({
                    name: this.form.get('name').value,
                    tags: this.form.get('tags').value
                });
            } else {
                const data = {
                    ...this.group,
                    info: {
                        ...this.group.info,
                        name: {
                            ru: this.form.get('name').value,
                            en: this.form.get('name').value
                        },
                        tags: this.form.get('tags').value
                    }
                };

                if (this.isCopy) {
                    this.groupsFacade.copyGroup(data, this.redirect);
                } else {
                    this.groupsFacade.updateGroup(data);
                }
            }
        });
    }

    cancel() {
        this.modal.close();
    }
}
