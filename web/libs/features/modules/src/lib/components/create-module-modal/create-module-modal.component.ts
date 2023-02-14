import { Component, Inject, OnDestroy, OnInit, TemplateRef, ViewChild } from '@angular/core';
import {
    AbstractControl,
    AsyncValidatorFn,
    NgForm,
    ValidationErrors,
    Validators,
    FormControl,
    FormGroup
} from '@angular/forms';
import { Router } from '@angular/router';
import { TranslocoService } from '@ngneat/transloco';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { McSidepanelPosition, McSidepanelService } from '@ptsecurity/mosaic/sidepanel';
import { filter, first, from, map, Observable, Subject, switchMap, takeUntil, toArray } from 'rxjs';

import {
  ModelsModuleInfo,
  ModelsModuleInfoOS,
  ModelsOptionsActions,
  ModelsOptionsEvents,
  ModelsOptionsTags,
  ModuleTemplate
} from '@soldr/api';
import { PERMISSIONS_TOKEN } from '@soldr/core';
import { convertVersion } from '@soldr/features/modules';
import { ENTITY_NAME_MAX_LENGTH, ListItem, ModelsFormControl, moduleOsList, ProxyPermission } from '@soldr/shared';
import { ModuleListFacade } from '@soldr/store/modules';
import { SharedFacade } from '@soldr/store/shared';

import { tagNameMask, tagNameMaskForReplace } from '../../utils/constants';

const MODULE_TEMPLATE = [
    ModuleTemplate.Empty,
    ModuleTemplate.Generic,
    ModuleTemplate.Collector,
    ModuleTemplate.Detector,
    ModuleTemplate.Responder
];
const MIN_LENGTH = 3;
const DEFAULT_VERSION = '0.1.0';

const regexTagFields = /^[a-zA-Z\d_\.]+$/g;
const regexTagFieldsForReplace = /[^a-zA-Z\d_\.]+/g;
const regexTags = /^[a-zA-Z\d_]+$/g;
const regexTagsForReplace = /[^a-zA-Z\d_]+/g;

@Component({
    selector: 'soldr-create-module-modal',
    templateUrl: './create-module-modal.component.html',
    styleUrls: ['./create-module-modal.component.scss']
})
export class CreateModuleModalComponent implements OnInit, OnDestroy {
    @ViewChild('sidePanel', { static: false }) sidePanel: TemplateRef<any>;
    @ViewChild('editForm', { static: false }) editForm: NgForm;

    form: FormGroup<
        ModelsFormControl<Omit<ModelsModuleInfo, 'version' | 'os'>> & {
            version: FormControl<string>;
            os: FormGroup<{ [key: string]: FormControl<boolean> }>;
        }
    >;
    themePalette = ThemePalette;
    moduleTemplates: ListItem[] = [];
    isFailedCreateModule = false;
    errorText: string;
    osList: ListItem[] = [...moduleOsList];
    actionList: string[] = [];
    eventList: string[] = [];
    fieldList: string[] = [];
    tagNameMask = tagNameMask;
    tagNameMaskForReplace = tagNameMaskForReplace;
    regexTagFields = regexTagFields;
    regexTagFieldsForReplace = regexTagFieldsForReplace;
    regexTags = regexTags;
    regexTagsForReplace = regexTagsForReplace;

    tags$ = this.sharedFacade.optionsTags$.pipe(map((tags) => this.setOptions(tags)));

    readonly destroyed$: Subject<void> = new Subject();

    constructor(
        private sidepanelService: McSidepanelService,
        private translocoService: TranslocoService,
        private modulesFacade: ModuleListFacade,
        private sharedFacade: SharedFacade,
        private router: Router,
        @Inject(PERMISSIONS_TOKEN) private permitted: ProxyPermission
    ) {}

    ngOnInit() {
        this.moduleTemplates = MODULE_TEMPLATE.map(
            (template): ListItem => ({
                value: template,
                label: this.translocoService.translate(
                    `modules.Modules.CreateModule.SelectItem.${template[0].toUpperCase()}${template.slice(1)}`
                )
            })
        );

        this.sharedFacade.fetchActions();
        this.sharedFacade.fetchEvents();
        this.sharedFacade.fetchFields();
        this.sharedFacade.fetchTags();

        this.modulesFacade.createdModule$
            .pipe(
                filter((module) => Boolean(module)),
                takeUntil(this.destroyed$)
            )
            .subscribe((module) => {
                this.sidepanelService.closeAll();
                if (this.permitted.EditModules) {
                    const route = ['/modules', module.info.name, 'edit'];
                    this.router.navigate(route);
                }
                this.form.reset();
            });
        this.modulesFacade.createError$.pipe(filter(Boolean), takeUntil(this.destroyed$)).subscribe((error) => {
            this.errorText = this.translocoService.translate(`modules.${error.code}.Error`);
            if (!this.errorText) {
                this.errorText = this.translocoService.translate('modules.Modules.CreateModule.InvalidInfo.Error');
            }
            this.isFailedCreateModule = true;
        });
    }

    ngOnDestroy(): void {
        this.destroyed$.next();
        this.destroyed$.complete();
    }

    openSidePanel() {
        this.isFailedCreateModule = false;
        this.form = new FormGroup({
            name: new FormControl<string>(
                '',
                [
                    Validators.required,
                    Validators.minLength(MIN_LENGTH),
                    Validators.maxLength(ENTITY_NAME_MAX_LENGTH),
                    Validators.pattern('^[0-9a-z_]+$')
                ],
                [this.existNameValidator()]
            ),
            template: new FormControl<string>(ModuleTemplate.Generic),
            version: new FormControl<string>(DEFAULT_VERSION, [
                Validators.required,
                Validators.pattern('^\\d{1,2}\\.\\d{1,2}\\.\\d{1,3}$')
            ]),
            os: new FormGroup<{ [key: string]: FormControl<boolean> }>({
                ['windows:386']: new FormControl(true),
                ['windows:amd64']: new FormControl(true),
                ['linux:386']: new FormControl(true),
                ['linux:amd64']: new FormControl(true),
                ['darwin:amd64']: new FormControl(true)
            }),
            fields: new FormControl<string[]>([]),
            events: new FormControl<string[]>([], [], [this.eventNameExistsValidator()]),
            actions: new FormControl<string[]>([], [], [this.actionNameExistsValidator()]),
            tags: new FormControl<string[]>([])
        });
        this.sidepanelService.open(this.sidePanel, {
            position: McSidepanelPosition.Right,
            hasBackdrop: true
        });
    }

    closeSidePanel() {
        this.sidepanelService.closeAll();
    }

    saveModule() {
        setTimeout(() => {
            if (this.form.invalid) {
                return;
            }

            this.isFailedCreateModule = false;
            const formData = this.form.getRawValue();
            const os = formData.os as Record<string, boolean>;
            const isEmptyOs = Object.values(os).findIndex((osValue) => osValue) === -1;
            if (isEmptyOs) {
                Object.keys(os).forEach((os) => (formData.os[os] = true));
            }

            const moduleOS: ModelsModuleInfoOS = Object.keys(os).reduce(
                (acc: Record<string, string[]>, cur: string) => {
                    if (formData.os[cur]) {
                        const item = cur.split(':');
                        acc[item[0]] = acc[item[0]] || [];
                        acc[item[0]].push(item[1]);
                    }

                    return acc;
                },
                {}
            );

            const module: ModelsModuleInfo = {
                ...formData,
                version: convertVersion(formData.version),
                os: moduleOS
            };

            this.modulesFacade.createModule(module);
        });
    }

    private existNameValidator(): AsyncValidatorFn {
        this.modulesFacade.fetchModules();

        return (control: AbstractControl): Observable<ValidationErrors | null> =>
            this.modulesFacade.getIsModuleNameExists(control.value as string).pipe(
                first(),
                map((exists) => (exists ? { entityNameExists: true } : null))
            );
    }

    private setOptions(options: ModelsOptionsTags[]): string[] {
        return [...new Set(options.map(({ name }: { name: string }) => name))].sort();
    }

    private eventNameExistsValidator(): AsyncValidatorFn {
        return (control: AbstractControl): Observable<ValidationErrors | null> =>
            this.sharedFacade.optionsEvents$.pipe(
                first(),
                switchMap((events: ModelsOptionsEvents[]) => from(events)),
                filter((event: ModelsOptionsEvents) => control.value.find((name: string) => name === event.name)),
                toArray(),
                map((found: ModelsOptionsEvents[]) => (found.length > 0 ? { eventNameExists: true } : null))
            );
    }

    private actionNameExistsValidator(): AsyncValidatorFn {
        return (control: AbstractControl): Observable<ValidationErrors | null> =>
            this.sharedFacade.optionsActions$.pipe(
                first(),
                switchMap((actions: ModelsOptionsActions[]) => from(actions)),
                filter((action: ModelsOptionsActions) => control.value.find((name: string) => name === action.name)),
                toArray(),
                map((found: ModelsOptionsActions[]) => (found.length > 0 ? { actionNameExists: true } : null))
            );
    }
}
