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
import { combineLatest, combineLatestWith, filter, first, map, Observable, pairwise, Subscription } from 'rxjs';

import { ErrorResponse } from '@soldr/api';
import { Policy } from '@soldr/models';
import { ENTITY_NAME_MAX_LENGTH, getCopyName, LanguageService, ModalInfoService } from '@soldr/shared';
import { PoliciesFacade } from '@soldr/store/policies';
import { SharedFacade } from '@soldr/store/shared';
import { TagDomain, TagsFacade } from '@soldr/store/tags';

@Component({
    selector: 'soldr-edit-policy-modal',
    templateUrl: './edit-policy-modal.component.html',
    styleUrls: ['./edit-policy-modal.component.scss']
})
export class EditPolicyModalComponent implements OnInit, OnDestroy {
    @Input() policy?: Policy;
    @Input() isCopy?: boolean;
    @Input() redirect?: boolean;

    @Output() afterSave = new EventEmitter();

    @ContentChild(McButton) button: McButton;

    form: FormGroup<{ name: FormControl<string>; tags: FormControl<string[]> }>;
    isProcessingPolicy$: Observable<boolean>;
    isLoading$: Observable<boolean>;
    language$ = this.languageService.current$;
    localizedGroupsNames: string[];
    modal: McModalRef;
    allPoliciesNames$ = this.sharedFacade.allPolicies$.pipe(
        map((policies) => policies.reduce((acc, { info }) => [...acc, info.name.ru, info.name.en], []))
    );
    placeholder: string;
    subscription = new Subscription();
    createModalSubscription: Subscription;
    tags$ = this.tagsFacade.policiesTags$;
    themePalette = ThemePalette;

    constructor(
        private languageService: LanguageService,
        private modalInfoService: ModalInfoService,
        private modalService: McModalService,
        private sharedFacade: SharedFacade,
        private policiesFacade: PoliciesFacade,
        private tagsFacade: TagsFacade,
        private transloco: TranslocoService
    ) {}

    private loadDataToForm() {
        const localizedName = this.policy?.info.name[this.languageService.lang] || '';
        const name = this.isCopy
            ? getCopyName(
                  localizedName,
                  this.localizedGroupsNames,
                  this.transloco.translate('common.Common.Pseudo.Text.CopyPostfix')
              )
            : localizedName;

        this.form.patchValue({
            name,
            tags: [...(this.policy?.info.tags || [])]
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
            this.policiesFacade
                .getIsExistedPoliciesByName(control.value as string, exclude)
                .pipe(map((exists) => (exists ? { entityNameExists: true } : null)));
    }

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
    }

    ngOnInit(): void {
        this.initRx();

        const creatingSubscription = this.policiesFacade.isCreatingPolicy$
            .pipe(pairwise(), combineLatestWith(this.policiesFacade.createError$))
            .subscribe(([[oldValue, newValue], createError]) => {
                if (oldValue && !newValue) {
                    this.processPolicyActions(
                        createError,
                        this.transloco.translate('policies.Policies.EditPolicy.ErrorText.UpdatePolicy')
                    );
                }
            });
        this.subscription.add(creatingSubscription);

        const copyingSubscription = this.policiesFacade.isCopyingPolicy$
            .pipe(pairwise(), combineLatestWith(this.policiesFacade.copyError$))
            .subscribe(([[oldValue, newValue], copyError]) => {
                if (oldValue && !newValue) {
                    this.processPolicyActions(
                        copyError,
                        this.transloco.translate('policies.Policies.EditPolicy.ErrorText.CopyPolicy')
                    );
                }
            });
        this.subscription.add(copyingSubscription);

        const updatingSubscription = this.policiesFacade.isUpdatingPolicy$
            .pipe(pairwise(), combineLatestWith(this.policiesFacade.updateError$))
            .subscribe(([[oldValue, newValue], updateError]) => {
                if (oldValue && !newValue) {
                    this.processPolicyActions(
                        updateError,
                        this.transloco.translate('policies.Policies.EditPolicy.ErrorText.UpdatePolicy')
                    );
                }
            });
        this.subscription.add(updatingSubscription);
    }

    processPolicyActions(error: ErrorResponse, text?: string) {
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
        this.isLoading$ = combineLatest([
            this.sharedFacade.isLoadingAllPolicies$,
            this.tagsFacade.isLoadingPoliciesTags$
        ]).pipe(map(([isLoadingAllPolicies, isLoadingPoliciesTags]) => isLoadingAllPolicies || isLoadingPoliciesTags));

        this.isProcessingPolicy$ = combineLatest([
            this.policiesFacade.isCreatingPolicy$,
            this.policiesFacade.isCopyingPolicy$,
            this.policiesFacade.isUpdatingPolicy$
        ]).pipe(
            map(
                ([isCreatingPolicy, isUpdatingPolicy, isCopyingPolicy]) =>
                    isCreatingPolicy || isCopyingPolicy || isUpdatingPolicy
            )
        );

        const localizedPoliciesNamesSubscription = this.allPoliciesNames$.subscribe((v: string[]) => {
            this.localizedGroupsNames = v;
        });
        this.subscription.add(localizedPoliciesNamesSubscription);
    }

    open(content: TemplateRef<any>, footer: TemplateRef<any>) {
        if (!this.button.disabled) {
            this.sharedFacade.fetchAllPolicies();
            this.tagsFacade.fetchTags(TagDomain.Policies);

            this.defineForm();

            this.modal = this.modalService.create({
                mcTitle: this.policy
                    ? this.isCopy
                        ? this.transloco.translate('policies.Policies.EditPolicy.ModalTitle.CopyPolicy')
                        : this.transloco.translate('policies.Policies.EditPolicy.ModalTitle.EditPolicy')
                    : this.transloco.translate('policies.Policies.EditPolicy.ModalTitle.CreatePolicy'),
                mcContent: content,
                mcFooter: footer,
                mcSize: ModalSize.Small
            });

            this.createModalSubscription = this.isLoading$
                .pipe(
                    pairwise(),
                    filter(([oldValue, newValue]) => oldValue && !newValue)
                )
                .subscribe(() => {
                    this.loadDataToForm();
                });

            const afterCloseSubscription = this.modal.afterClose.subscribe(() =>
                this.createModalSubscription.unsubscribe()
            );
            this.subscription.add(afterCloseSubscription);
        }
    }

    save() {
        setTimeout(() => {
            if (!this.form.valid) {
                return;
            }

            if (!this.policy) {
                this.policiesFacade.createPolicy({
                    name: this.form.get('name').value,
                    tags: this.form.get('tags').value
                });
            } else {
                const data = {
                    ...this.policy,
                    info: {
                        ...this.policy.info,
                        name: {
                            ru: this.form.get('name').value,
                            en: this.form.get('name').value
                        },
                        tags: this.form.get('tags').value
                    }
                };

                if (this.isCopy) {
                    this.policiesFacade.copyPolicy(data, this.redirect);
                } else {
                    this.policiesFacade.updatePolicy(data);
                }
            }
        });
    }

    cancel() {
        this.modal.close();
    }
}
