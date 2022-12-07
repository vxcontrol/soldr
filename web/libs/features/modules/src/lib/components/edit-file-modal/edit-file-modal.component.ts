import {
    ChangeDetectorRef,
    Component,
    ElementRef,
    Input,
    OnDestroy,
    OnInit,
    TemplateRef,
    ViewChild
} from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { McModalRef, McModalService, ModalSize } from '@ptsecurity/mosaic/modal';
import { McToastService } from '@ptsecurity/mosaic/toast';
import * as monaco from 'monaco-editor';
import { combineLatestWith, map, of, pairwise, Subject, Subscription, switchMap, take } from 'rxjs';

import { ModelsModuleS } from '@soldr/api';
import { FilesContent } from '@soldr/models';
import { LanguageService, ModuleVersionPipe, OsDetectorService } from '@soldr/shared';
import { ModuleEditFacade } from '@soldr/store/modules';

import { ModuleFolderSection } from '../../types';
import { supportedLanguages } from '../../utils';
import { CreateFileModalComponent } from '../create-file-modal/create-file-modal.component';

interface Tab {
    content: string;
    extension: string;
    filename: string;
    filepath: string;
    loaded?: boolean;
    prefix: string;
}

@Component({
    selector: 'soldr-edit-file-modal',
    templateUrl: './edit-file-modal.component.html',
    styleUrls: ['./edit-file-modal.component.scss']
})
export class EditFileModalComponent implements OnInit, OnDestroy {
    @Input() files: string[] = [];
    @Input() module: ModelsModuleS;
    @Input() readOnly: boolean;

    @ViewChild('header') header: TemplateRef<any>;
    @ViewChild('content') content: TemplateRef<any>;
    @ViewChild('footer') footer: TemplateRef<any>;

    activeTabIndex = -1;
    canUpdateModuleInPolicies$ = this.moduleEditFacade.canUpdateModuleInPolicies$;
    filterText = '';
    isLoadingFiles$ = this.moduleEditFacade.isLoadingFiles$;
    isSavingFiles$ = this.moduleEditFacade.isSavingFiles$;
    isUpdatingModuleInPolicies$ = this.moduleEditFacade.isUpdatingModuleInPolicies$;
    lang$ = this.languageService.current$;
    modal: McModalRef;
    module$ = this.moduleEditFacade.module$;
    openedFiles$ = this.moduleEditFacade.openedFiles$;
    openedFiles: FilesContent;
    openedTabs: Tab[] = [];
    options: any = {};
    supportedLanguages = supportedLanguages;

    private subscription = new Subscription();

    private static getPrefixTab(filepath: string) {
        const sectionPosition = 2;

        switch (filepath.split('/')[sectionPosition]) {
            case ModuleFolderSection.Agent:
                return 'A';
            case ModuleFolderSection.Server:
                return 'S';
            case ModuleFolderSection.Browser:
                return 'B';
            default:
                return '';
        }
    }

    private onBeforeUnload = (event: BeforeUnloadEvent) => {
        if (this.changedTabs.length > 0) {
            event.preventDefault();
            event.returnValue = '';

            return '';
        }

        return false;
    };

    private onKeyDown = (event: KeyboardEvent) => {
        const shouldSave = (event.ctrlKey || (!event.ctrlKey && event.metaKey)) && event.code === 'KeyS';
        const shouldUpdate = (event.ctrlKey || (!event.ctrlKey && event.metaKey)) && event.code === 'KeyD';
        const shouldRefresh = (event.ctrlKey || (!event.ctrlKey && event.metaKey)) && event.code === 'KeyJ';

        if (shouldSave) {
            event.preventDefault();
            this.saveFileInTab(this.activeTab);
        }

        if (shouldUpdate) {
            event.preventDefault();
            this.updateInPolicies();
        }

        if (shouldRefresh) {
            event.preventDefault();
            this.synchronize();
        }
    };

    constructor(
        private element: ElementRef,
        private moduleEditFacade: ModuleEditFacade,
        private toastService: McToastService,
        private transloco: TranslocoService,
        private cdr: ChangeDetectorRef,
        private languageService: LanguageService,
        private modalService: McModalService,
        public os: OsDetectorService
    ) {}

    ngOnInit(): void {
        const openedFilesSubscription = this.openedFiles$.pipe(pairwise()).subscribe(([oldData, data]) => {
            this.openedFiles = data;

            for (const path of Object.keys(data)) {
                const existedTab = this.openedTabs.find((tab) => path === tab.filepath);

                if (existedTab && !oldData[path]?.loaded && data[path].loaded) {
                    existedTab.content = data[path].content;
                    setTimeout(() => {
                        this.initEditor(existedTab);
                    });
                }
            }
        });
        this.subscription.add(openedFilesSubscription);

        const savingErrorsSubscription = this.moduleEditFacade.saveFileErrors$.subscribe((errors) => {
            for (const unsavedFilePath of Object.keys(errors)) {
                this.toastService.show({
                    style: 'error',
                    title: this.transloco.translate('modules.Modules.ModuleEdit.ErrorText.SaveFileError', {
                        filename: unsavedFilePath
                    })
                });
            }
        });
        this.subscription.add(savingErrorsSubscription);

        const loadingErrorsSubscription = this.moduleEditFacade.loadFileErrors$.subscribe((errors) => {
            for (const notLoadedFile of Object.keys(errors as object)) {
                this.toastService.show({
                    style: 'error',
                    title: this.transloco.translate('modules.Modules.ModuleEdit.ErrorText.LoadFileError', {
                        filename: notLoadedFile
                    })
                });
                this.removeTabByPath(notLoadedFile);
            }
        });
        this.subscription.add(loadingErrorsSubscription);

        window.addEventListener('beforeunload', this.onBeforeUnload, { capture: true });
        window.addEventListener('keydown', this.onKeyDown, { capture: true });
    }

    get activeTab() {
        return this.openedTabs[this.activeTabIndex];
    }

    get changedTabs() {
        return this.openedTabs.filter(
            (tab) => tab.content !== this.openedFiles[tab.filepath].content && this.openedFiles[tab.filepath].loaded
        );
    }

    get root() {
        return `${this.module.info.name}/${new ModuleVersionPipe().transform(this.module.info.version)}`;
    }

    get canSave() {
        return (
            this.activeTab &&
            this.openedFiles[this.activeTab.filepath]?.loaded &&
            this.activeTab.content !== this.openedFiles[this.activeTab.filepath]?.content
        );
    }

    ngOnDestroy(): void {
        window.removeEventListener('beforeunload', this.onBeforeUnload);
        window.removeEventListener('keydown', this.onKeyDown);
        this.subscription.unsubscribe();
    }

    open(filepath: string) {
        this.modal = this.modalService.create({
            mcCloseByESC: false,
            mcTitle: this.header,
            mcStyle: {
                background: 'red'
            },
            mcBodyStyle: {
                height: '100vh',
                'max-height': 'calc(100vh - var(--mc-modal-header-size-height))',
                padding: 0
            },
            mcWidth: '100vw',
            mcContent: this.content,
            mcFooter: this.footer,
            mcOnCancel: () => {
                of(this.changedTabs.length)
                    .pipe(
                        switchMap((unsavedFilesCount) => {
                            if (unsavedFilesCount > 0) {
                                return this.confirmCloseUnsavedFiles();
                            } else {
                                return of(false);
                            }
                        }),
                        switchMap((confirmed) => {
                            if (confirmed) {
                                const requestForSaving = this.changedTabs.map((tab) => ({
                                    path: tab.filepath,
                                    content: tab.content
                                }));

                                this.moduleEditFacade.saveFiles(requestForSaving);

                                return this.moduleEditFacade.isSavingFiles$.pipe(
                                    pairwise(),
                                    take(2),
                                    combineLatestWith(this.moduleEditFacade.saveFileErrors$),
                                    map(
                                        ([[oldValue, newValue], errors]) =>
                                            oldValue && !newValue && Object.keys(errors || {}).length === 0
                                    )
                                );
                            } else {
                                return of(true);
                            }
                        })
                    )
                    .subscribe({
                        next: (result) => {
                            if (result) {
                                this.modal.close();
                            }
                        }
                    });

                return false;
            }
        });

        this.modal.afterOpen.pipe(take(1)).subscribe(() => {
            this.modal.getElement().querySelector<HTMLElement>('.mc-modal').style.top = '0px';
            this.modal.getElement().querySelector<HTMLElement>('.mc-modal-header').style.boxSizing = 'border-box';
            this.modal
                .getElement()
                .querySelector<HTMLElement>('.mc-modal-header')
                .classList.add('edit-file-modal__header');
            this.modal.getElement().querySelector<HTMLElement>('.mc-modal-title').classList.add('flex-auto');
            this.modal
                .getElement()
                .querySelector<HTMLElement>('.mc-modal-title')
                .classList.add('layout-padding-right-4xl');
        });

        this.modal.afterClose.pipe(take(1)).subscribe(() => {
            this.filterText = '';
            this.openedTabs = [];
            this.moduleEditFacade.resetFile();
        });

        this.openFile(filepath);
    }

    openFile(filepath: string) {
        const tabIndex = this.openedTabs.findIndex((tab) => tab.filepath === filepath);
        if (tabIndex !== -1) {
            this.activeTabIndex = tabIndex;
        } else {
            this.moduleEditFacade.loadFiles([filepath]);
            this.openedTabs.push(this.initTab(filepath));
            this.activeTabIndex = this.openedTabs.length - 1;
        }
    }

    closeTab(tab: Tab, confirm?: boolean) {
        if (confirm) {
            this.confirmCloseUnsavedFiles(tab.filepath).subscribe((confirmed) => {
                if (confirmed) {
                    this.saveFileInTab(tab);
                } else {
                    this.moduleEditFacade.closeFiles([tab.filepath]);
                    this.removeTabByPath(tab.filepath);
                }
            });
        } else {
            this.moduleEditFacade.closeFiles([tab.filepath]);
            this.removeTabByPath(tab.filepath);
        }
    }

    saveFileInTab(tab: Tab) {
        this.moduleEditFacade.saveFiles([{ path: tab.filepath, content: tab.content }]);
    }

    confirmCloseUnsavedFiles(filename?: string) {
        const result = new Subject<boolean>();

        const modal: McModalRef<CreateFileModalComponent> = this.modalService.create({
            mcSize: ModalSize.Small,
            mcContent: filename
                ? this.transloco.translate('modules.Modules.ModuleEdit.Text.CloseModifiedFile', { filename })
                : this.transloco.translate('modules.Modules.ModuleEdit.Text.CloseModifiedFiles'),
            mcFooter: [
                {
                    label: this.transloco.translate('common.Common.Pseudo.ButtonText.Save'),
                    type: ThemePalette.Primary,
                    mcModalMainAction: true,
                    onClick: () => {
                        result.next(true);
                        result.complete();
                        modal.close();
                    }
                },
                {
                    label: this.transloco.translate('common.Common.Pseudo.ButtonText.DontSave'),
                    type: ThemePalette.Default,
                    onClick: () => {
                        result.next(false);
                        result.complete();
                        modal.close();
                    }
                },
                {
                    label: this.transloco.translate('common.Common.Pseudo.ButtonText.Cancel'),
                    type: ThemePalette.Default,
                    onClick: () => {
                        result.complete();
                        modal.close();
                    }
                }
            ]
        });

        return result;
    }

    updateInPolicies() {
        this.moduleEditFacade.updateModuleInPolicies(
            this.module.info.name,
            new ModuleVersionPipe().transform(this.module.info.version)
        );
    }

    synchronize() {
        this.moduleEditFacade.loadFiles(this.openedTabs.map(({ filepath }) => filepath));
    }

    private removeTabByPath(filepath: string) {
        this.openedTabs = this.openedTabs.filter((item) => filepath !== item.filepath);
    }

    private initTab(path: string, content?: string) {
        return {
            filename: path.split('/').slice(-1)[0],
            prefix: EditFileModalComponent.getPrefixTab(path),
            filepath: path,
            extension: path.split('.').splice(-1)[0],
            content
        } as Tab;
    }

    private initEditor(tab: Tab) {
        const editorContainer: HTMLElement = this.modal.getElement().querySelector(`.edit-file-modal__editor`);
        const editor = monaco.editor.create(editorContainer, {
            automaticLayout: true,
            readOnly: this.readOnly,
            language: supportedLanguages[tab.extension] || '',
            unicodeHighlight: {
                ambiguousCharacters: false
            },
            value: tab.content,
            theme: 'soldrFileTheme'
        });
        editor.onDidChangeModelContent(() => {
            tab.content = editor.getValue();
        });
    }
}
