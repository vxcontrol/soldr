import { Component, EventEmitter, OnDestroy, OnInit, Output, TemplateRef, ViewChild } from '@angular/core';
import { NgForm, Validators, FormGroup, FormControl } from '@angular/forms';
import { TranslocoService } from '@ngneat/transloco';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { McModalRef, McModalService, ModalSize } from '@ptsecurity/mosaic/modal';
import * as JSZip from 'jszip';
import { DateTime } from 'luxon';
import {
    catchError,
    concatMap,
    filter,
    first,
    forkJoin,
    from,
    last,
    map,
    Observable,
    of,
    pairwise,
    scan,
    Subject,
    takeUntil
} from 'rxjs';

import { ErrorResponse, ModelsChangelog, ModelsLocale } from '@soldr/api';
import { LanguageService } from '@soldr/shared';
import { ModuleListFacade } from '@soldr/store/modules';

const DATE_ERROR = 'Invalid format of datetime in changelog';
const REGEX_PARSE_MODULE =
    /(?<fileName>^[^\/\\:*?<>|"]+)(\/|.*)\/(?<version>\d+\.\d+\.\d+).*\/(locale|changelog)\.json/;

@Component({
    selector: 'soldr-import-module-modal',
    templateUrl: './import-module-modal.component.html',
    styleUrls: ['./import-module-modal.component.scss']
})
export class ImportModuleModalComponent implements OnInit, OnDestroy {
    @Output() afterImport = new EventEmitter<void>();

    @ViewChild('tplTitle', { static: false }) tplTitle: TemplateRef<any>;
    @ViewChild('tplContent', { static: false }) tplContent: TemplateRef<any>;
    @ViewChild('tplFooter', { static: false }) tplFooter: TemplateRef<any>;
    @ViewChild('editForm', { static: false }) editForm: NgForm;

    defaultVersion = 'all';
    errorImportTextSecond: string;
    errorImportText: string;
    form: FormGroup<{ name: FormControl<string>; version: FormControl<string>; isRewrite: FormControl<boolean> }>;
    importFailed = false;
    importInProgress = false;
    selectedFile: File;
    themePalette = ThemePalette;
    tplModal: McModalRef;
    value: string;

    options$: Observable<{ modulesByTitle: Map<string, string>; versionsByName: Map<string, string[]> }>;

    readonly destroyed$: Subject<void> = new Subject();

    constructor(
        private languageService: LanguageService,
        private modalService: McModalService,
        private modulesFacade: ModuleListFacade,
        private translocoService: TranslocoService
    ) {}

    ngOnInit() {
        this.modulesFacade.importError$
            .pipe(filter(Boolean), takeUntil(this.destroyed$))
            .subscribe((error) => this.setError(error));
        this.modulesFacade.isImportingModule$
            .pipe(pairwise(), takeUntil(this.destroyed$))
            .subscribe(([previous, current]) => {
                this.importInProgress = current;
                if (previous && !current && !this.importFailed) {
                    this.importSuccess();
                }
            });
    }

    ngOnDestroy(): void {
        this.destroyed$.next();
        this.destroyed$.complete();
    }

    onFileSelected(event: Event) {
        this.parseFile((event.target as HTMLInputElement).files[0]);
    }

    initForm() {
        this.form = new FormGroup({
            name: new FormControl<string>('', Validators.required),
            version: new FormControl<string>(this.defaultVersion),
            isRewrite: new FormControl<boolean>(false)
        });
    }

    openModal() {
        this.tplModal = this.modalService.create({
            mcSize: ModalSize.Small,
            mcTitle: this.tplTitle,
            mcContent: this.tplContent,
            mcFooter: this.tplFooter,
            mcOnCancel: () => this.destroyModal()
        });
    }

    importModule() {
        setTimeout(() => {
            if (this.form.invalid) {
                return;
            }

            if (this.selectedFile) {
                const formData = this.form.getRawValue();
                const data = new FormData();
                data.append('archive', this.selectedFile);
                this.modulesFacade.importModule(formData.name, formData.version, {
                    rewrite: formData.isRewrite,
                    archive: data
                });
            }
        });
    }

    setError(error: ErrorResponse) {
        this.importFailed = true;
        const errorMsg = /\.([a-zA-Z]+)$/.exec(error.code)[1];
        this.errorImportText = this.translocoService.translate('modules.Modules.ImportModule.ErrorText.LoadingFailed');
        this.errorImportTextSecond = this.translocoService.translate(
            `modules.Modules.ImportModule.ErrorText.${errorMsg}`
        );
    }

    setParseError(error: any) {
        this.importFailed = true;
        this.errorImportText =
            error.message === DATE_ERROR
                ? error.message
                : this.translocoService.translate('modules.Modules.ImportModule.ErrorText.ParserFail');
    }

    parseFile(file: File) {
        this.selectedFile = file;

        const zip = new JSZip();
        const modulesData$ = from(zip.loadAsync(file)).pipe(
            concatMap((zipFile) =>
                from(this.getModulePaths(zipFile)).pipe(
                    map((path: string) => ({
                        path,
                        name: path.split('/')[0]
                    })),
                    concatMap((data) =>
                        forkJoin([
                            zipFile.file(`${data.path}/config/locale.json`).async('text'),
                            zipFile.file(`${data.path}/config/changelog.json`).async('text')
                        ]).pipe(
                            map(([locale, changelog]) => ({
                                ...data,
                                locale,
                                changelog
                            }))
                        )
                    )
                )
            )
        );
        this.options$ = modulesData$.pipe(
            scan(
                (acc, data) => {
                    const changelog = JSON.parse(data.changelog) as ModelsChangelog;
                    const locale = JSON.parse(data.locale) as ModelsLocale;
                    const previousVersions = acc.versionsByName.get(data.name);
                    const currentVersions = Object.keys(changelog);
                    const title = locale.module[this.languageService.lang].title;

                    this.validateDate(currentVersions, changelog);

                    acc.modulesByTitle.set(title, data.name);
                    acc.versionsByName.set(data.name, [...new Set([...(previousVersions || []), ...currentVersions])]);

                    return acc;
                },
                { modulesByTitle: new Map<string, string>(), versionsByName: new Map<string, string[]>() }
            ),
            last(),
            catchError((error) => {
                this.setParseError(error);

                return of(error);
            })
        );

        this.options$.subscribe(() => {
            this.initForm();
            this.openModal();
        });
    }

    importSuccess() {
        this.destroyModal();
        this.afterImport.emit();
    }

    destroyModal() {
        this.tplModal.destroy();
        this.tplModal.afterClose.pipe(first()).subscribe(() => {
            this.importFailed = false;
            this.value = undefined;
            this.errorImportText = '';
            this.errorImportTextSecond = '';
            this.modulesFacade.resetImportState();
        });
    }

    private getModulePaths(zipFile: JSZip) {
        const modulePaths = new Set<string>();

        Object.keys(zipFile.files).forEach((path: string) => {
            const match = REGEX_PARSE_MODULE.exec(path);
            if (match) {
                modulePaths.add(`${match.groups.fileName}/${match.groups.version}`);
            }
        });
        if (!modulePaths.size) {
            throw new Error();
        }

        return modulePaths;
    }

    private validateDate(versions: string[], changelog: ModelsChangelog) {
        const isValidDates = versions.every(
            (version) =>
                DateTime.fromFormat(changelog[version].ru.date, 'dd.MM.yyyy').isValid &&
                DateTime.fromFormat(changelog[version].en.date, 'MM-dd-yyyy').isValid
        );
        if (!isValidDates) {
            throw new Error(DATE_ERROR);
        }
    }
}
