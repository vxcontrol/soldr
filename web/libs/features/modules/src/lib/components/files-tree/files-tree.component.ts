import {
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    OnChanges,
    OnDestroy,
    OnInit,
    Output,
    SimpleChanges
} from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';
import { McModalRef, McModalService, ModalSize } from '@ptsecurity/mosaic/modal';
import { McToastService } from '@ptsecurity/mosaic/toast';
import { defaultCompareValues, FlatTreeControl, McTreeFlatDataSource, McTreeFlattener } from '@ptsecurity/mosaic/tree';
import * as base64 from 'base64-js';
import {
    BehaviorSubject,
    combineLatest,
    filter,
    from,
    fromEvent,
    map,
    of,
    race,
    Subject,
    Subscription,
    switchMap,
    take,
    throwError,
    toArray
} from 'rxjs';

import {
    ModelsModuleS,
    ModulesService,
    PrivateSystemModuleFile,
    PrivateSystemModuleFilePatch,
    StatusResponse,
    SuccessResponse
} from '@soldr/api';
import { supportedLanguages } from '@soldr/features/modules';
import { ModuleVersionPipe } from '@soldr/shared';
import { ModuleEditFacade } from '@soldr/store/modules';

import { FilesTreeService } from '../../services';
import { FilesTreeNode, FileTreeNodeType, ModuleFolderSection } from '../../types';
import { CreateFileModalComponent } from '../create-file-modal/create-file-modal.component';
import { DeleteFileModalComponent } from '../delete-file-modal/delete-file-modal.component';
import { MoveFileModalComponent } from '../move-file-modal/move-file-modal.component';

@Component({
    selector: 'soldr-files-tree',
    templateUrl: './files-tree.component.html',
    styleUrls: ['./files-tree.component.scss']
})
export class FilesTreeComponent implements OnInit, OnChanges, OnDestroy {
    @Input() files: string[];
    @Input() filter: string;
    @Input() hideClibs: boolean;
    @Input() moduleName: string;
    @Input() moduleVersion: string;
    @Input() readOnly: boolean;
    @Input() section: ModuleFolderSection;

    @Output() openFile = new EventEmitter<string>();

    staticBackendFiles = ['main.lua', 'args.json'];
    staticBrowserFiles = ['main.vue'];
    filesDataSource: McTreeFlatDataSource<FilesTreeNode, FilesTreeNode>;
    filesTreeControl: FlatTreeControl<FilesTreeNode>;
    filesTreeFlattener: McTreeFlattener<FilesTreeNode, FilesTreeNode>;
    selectedItem: string;
    module$ = this.moduleEditFacade.module$;
    files$ = new BehaviorSubject<string[]>([]);
    normalizedFiles$ = this.files$.pipe(
        switchMap((paths) =>
            from(paths).pipe(
                map((path) => path.split('/').splice(2).join('/')),
                toArray()
            )
        )
    );
    FileTreeNodeType = FileTreeNodeType;

    private arch = ['386', 'amd64'];
    private defaultClibsTree: string[];
    private os = ['windows', 'darwin', 'linux'];
    private subscription = new Subscription();

    constructor(
        private moduleEditFacade: ModuleEditFacade,
        private modulesService: ModulesService,
        private toastService: McToastService,
        private transloco: TranslocoService,
        private modalService: McModalService,
        private filesTreeService: FilesTreeService,
        private cdr: ChangeDetectorRef
    ) {
        this.initTree();
        this.defaultClibsTree = this.getClibsDefaultTree();
    }

    get root() {
        return `${this.moduleName}/${this.moduleVersion}`;
    }

    ngOnInit(): void {
        const filesSubscription = this.normalizedFiles$.subscribe((files) => {
            this.filesDataSource.data = this.filesTreeService.buildTree(files, this.hideClibs, this.section);
            this.filesTreeControl.expandAll();
        });

        this.subscription.add(filesSubscription);
    }

    ngOnChanges({ files, filter }: SimpleChanges): void {
        if (files?.currentValue) {
            this.files$.next(this.files);
        }

        if (filter) {
            this.filesTreeControl.filterNodes(this.filter);

            if (!filter.currentValue && filter.previousValue) {
                this.filesTreeControl.expandAll();
            }

            this.cdr.detectChanges();
        }
    }

    ngOnDestroy(): void {
        this.subscription.unsubscribe();
    }

    hasChild(index: number, node: FilesTreeNode) {
        return [FileTreeNodeType.Section, FileTreeNodeType.Part, FileTreeNodeType.Folder].includes(node.type);
    }

    downloadFile(module: ModelsModuleS, path: string) {
        const processedPath = `${this.root}/${this.restoreCodePath(path)}`;
        const { name, version } = module.info;

        this.modulesService.fetchFile(name, new ModuleVersionPipe().transform(version), processedPath).subscribe({
            next: (response: SuccessResponse<PrivateSystemModuleFile>) => {
                if (response.status === StatusResponse.Success) {
                    const blob = new Blob([base64.toByteArray(response.data?.data)], {
                        type: 'octet/stream'
                    });
                    const link = document.createElement('a');
                    link.href = URL.createObjectURL(blob);
                    link.download = path.split('/').splice(-1)[0];
                    link.click();
                    URL.revokeObjectURL(link.href);
                } else {
                    this.showDownloadError();
                }
            },
            error: () => {
                this.showDownloadError();
            }
        });
    }

    uploadFile(module: ModelsModuleS, folderPath: string) {
        const processedPath = folderPath.replace(/^code\/?/, '');
        const input = document.createElement('input');
        const { name, version } = module.info;
      const transformedVersion = new ModuleVersionPipe().transform(version);

        input.type = 'file';

        const p = fromEvent(input, 'input').pipe(
            take(1),
            map(() => Array.from(input.files)[0]),
            switchMap((file: File) => combineLatest([of(file), this.readFile(file)])),
            switchMap(([file, data]: [File, string]) =>
                combineLatest([
                    this.selectPathForUpload(this.root, `${processedPath ? `${processedPath}/` : ''}${file.name}`),
                    of(data)
                ])
            ),
            switchMap(([path, data]: [string, string]) => {
                const params: PrivateSystemModuleFilePatch = {
                    action: 'save',
                    data,
                    path: `${this.root}/${path}`
                };

                return this.modulesService.patchFile(name, transformedVersion, params);
            })
        );

        p.subscribe({
            next: () => {
              this.moduleEditFacade.fetchFiles(name, transformedVersion);
              this.moduleEditFacade.fetchUpdates(name, transformedVersion);
            },
            error: () => this.showUploadError()
        });

        input.click();
    }

    deleteObject(module: ModelsModuleS, path: string) {
        const processedPath = `${this.root}/${path.replace(/^code\//, '')}`;
        const { name, version } = module.info;
        const transformedVersion = new ModuleVersionPipe().transform(version);

        this.warnAboutDeleting(path)
            .pipe(
                filter(Boolean),
                switchMap(() => {
                    const params: PrivateSystemModuleFilePatch = {
                        action: 'remove',
                        path: processedPath
                    };

                    return this.modulesService.patchFile(name, transformedVersion, params);
                })
            )
            .subscribe({
                next: (response: SuccessResponse<PrivateSystemModuleFile>) => {
                    if (response.status === StatusResponse.Success) {
                        this.moduleEditFacade.fetchFiles(name, transformedVersion);
                        this.moduleEditFacade.fetchUpdates(name, transformedVersion);
                    } else {
                        this.showDeleteError();
                    }
                },
                error: () => {
                    this.showDeleteError();
                }
            });
    }

    createFile(module: ModelsModuleS, path: string) {
        const { name, version } = module.info;
        const transformedVersion = new ModuleVersionPipe().transform(version);

        this.selectPathForCreate(this.root, `${path}/new_file.ext`)
            .pipe(
                switchMap((filepath: string) => {
                    const params: PrivateSystemModuleFilePatch = {
                        action: 'save',
                        data: '',
                        path: `${this.root}/${this.restoreCodePath(filepath)}`
                    };

                    return this.modulesService.patchFile(name, new ModuleVersionPipe().transform(version), params);
                })
            )
            .subscribe({
                next: (response: SuccessResponse<PrivateSystemModuleFile>) => {
                    if (response.status === StatusResponse.Success) {
                        this.moduleEditFacade.fetchFiles(name, new ModuleVersionPipe().transform(version));
                        this.moduleEditFacade.fetchUpdates(name, transformedVersion);
                    } else {
                        this.showCreateError();
                    }
                },
                error: () => {
                    this.showCreateError();
                }
            });
    }

    moveObject(module: ModelsModuleS, path: string) {
        const { name, version } = module.info;
        const transformedVersion = new ModuleVersionPipe().transform(version);

        this.selectPathForMoving(this.root, path)
            .pipe(
                switchMap((newPath: string) => {
                    const params: PrivateSystemModuleFilePatch = {
                        action: 'move',
                        path: `${this.root}/${this.restoreCodePath(path)}`,
                        newpath: `${this.root}/${this.restoreCodePath(newPath)}`
                    };

                    return this.modulesService.patchFile(name, transformedVersion, params);
                })
            )
            .subscribe({
                next: (response: SuccessResponse<PrivateSystemModuleFile>) => {
                    if (response.status === StatusResponse.Success) {
                        this.moduleEditFacade.fetchFiles(name, transformedVersion);
                        this.moduleEditFacade.fetchUpdates(name, transformedVersion);
                    } else {
                        this.showMoveError();
                    }
                },
                error: () => {
                    this.showMoveError();
                }
            });
    }

    open(node: FilesTreeNode) {
        if (node?.type === FileTreeNodeType.File) {
            this.openFile.emit(`${this.root}/${this.restoreCodePath(node.data.path)}`);
        }
    }

    canEdit(node: FilesTreeNode) {
        const extension = node.data.path.split('.').slice(-1)[0];

        return !!supportedLanguages[extension];
    }

    canDeleteFile(node: FilesTreeNode) {
        return !this.getIsStaticFiles(node);
    }

    canMoveFile(node: FilesTreeNode) {
        return !this.getIsStaticFiles(node);
    }

    canDeleteFolder(node: FilesTreeNode) {
        return node.level > 0 && !this.getIsDefaultClibsFolders(node);
    }

    canMoveFolder(node: FilesTreeNode) {
        return node.level > 0 && !this.getIsDefaultClibsFolders(node);
    }

    getIsDefaultClibsFolders(node: FilesTreeNode) {
        return this.defaultClibsTree.some((path) => new RegExp(`^clibs/${path}$`).test(node.data.path));
    }

    getIsStaticFiles(node: FilesTreeNode) {
        return this.section === ModuleFolderSection.Browser
            ? this.staticBrowserFiles.some((path) => new RegExp(`^code/${path}$`).test(node.data.path))
            : this.staticBackendFiles.some((path) => new RegExp(`^code/${path}$`).test(node.data.path));
    }

    private initTree() {
        this.filesTreeFlattener = new McTreeFlattener(
            (node: FilesTreeNode) => node,
            this.getLevel,
            this.getIsExpandable,
            (node: FilesTreeNode) => node.children
        );
        this.filesTreeControl = new FlatTreeControl<FilesTreeNode>(
            this.getLevel,
            this.getIsExpandable,
            (node: FilesTreeNode) => node,
            (node: FilesTreeNode) => node.data.name,
            defaultCompareValues
        );

        this.filesDataSource = new McTreeFlatDataSource(this.filesTreeControl, this.filesTreeFlattener);
        this.filesDataSource.data = [];
    }

    private getLevel(node: FilesTreeNode) {
        return node.level;
    }

    private getIsExpandable(node: FilesTreeNode) {
        return (
            [FileTreeNodeType.Part, FileTreeNodeType.Section, FileTreeNodeType.Folder].includes(node.type) &&
            node.children?.length > 0
        );
    }

    private restoreCodePath(path: string) {
        return path?.replace(/^code\//, '');
    }

    private getClibsDefaultTree(maxLevel?: number) {
        const result = [];

        for (const osItem of this.os) {
            result.push(osItem);

            for (const archItem of this.arch) {
                if (maxLevel < 1) {
                    break;
                }

                if (archItem === '386' && osItem === 'darwin') {
                    continue;
                }

                result.push(`${osItem}/${archItem}`);

                if (maxLevel < 2) {
                    break;
                }

                result.push(`${osItem}/${archItem}/sys`);
            }
        }

        return result;
    }

    private readFile(file: File) {
        const reader = new FileReader();

        const readFile$ = race(
            fromEvent<ProgressEvent<FileReader>>(reader, 'load'),
            fromEvent(reader, 'error').pipe(switchMap(() => throwError(() => 'error')))
        ).pipe(
            map((event: ProgressEvent<FileReader>) => {
                if (event.target.readyState === 2) {
                    return (event.target.result as string).split('base64,')[1];
                }

                return undefined;
            })
        );

        readFile$.pipe(take(1)).subscribe({
            error: () => reader.abort()
        });

        reader.readAsDataURL(file);

        return readFile$;
    }

    private selectPathForUpload(prefix: string, filename: string) {
        const result = new Subject<string>();

        const modal: McModalRef<CreateFileModalComponent> = this.modalService.create({
            mcTitle: this.transloco.translate('modules.Modules.ModuleEdit.ModalTitle.UploadMsgTitle'),
            mcComponent: CreateFileModalComponent,
            mcComponentParams: {
                path: filename,
                prefix
            },
            mcOkText: this.transloco.translate('common.Common.Pseudo.ButtonText.Upload'),
            mcCancelText: this.transloco.translate('common.Common.Pseudo.ButtonText.Cancel'),
            mcOnOk: (instance: CreateFileModalComponent) => {
                const path = instance.getPath();

                if (path) {
                    result.next(path);
                    result.complete();
                    modal.close();
                }

                return false;
            },
            mcOnCancel: () => {
                result.complete();
                modal.close();
            }
        });

        return result;
    }

    private selectPathForMoving(prefix: string, path: string) {
        const result = new Subject<string>();

        const modal: McModalRef<MoveFileModalComponent> = this.modalService.create({
            mcTitle: this.transloco.translate('modules.Modules.ModuleEdit.ModalTitle.MoveMsgTitle'),
            mcComponent: MoveFileModalComponent,
            mcComponentParams: {
                oldPath: path,
                prefix
            },
            mcOkText: this.transloco.translate('common.Common.Pseudo.ButtonText.Move'),
            mcCancelText: this.transloco.translate('common.Common.Pseudo.ButtonText.Cancel'),
            mcOnOk: () => {
                result.next(modal.getContentComponent().path);
                result.complete();
                modal.close();
            },
            mcOnCancel: () => {
                result.complete();
                modal.close();
            }
        });

        return result;
    }

    private selectPathForCreate(prefix: string, filename: string) {
        const result = new Subject<string>();

        const modal: McModalRef<CreateFileModalComponent> = this.modalService.create({
            mcTitle: this.transloco.translate('modules.Modules.ModuleEdit.ModalTitle.CreateMsg'),
            mcComponent: CreateFileModalComponent,
            mcComponentParams: {
                path: filename,
                prefix
            },
            mcOkText: this.transloco.translate('common.Common.Pseudo.ButtonText.Create'),
            mcCancelText: this.transloco.translate('common.Common.Pseudo.ButtonText.Cancel'),
            mcOnOk: (instance: CreateFileModalComponent) => {
                const path = instance.getPath();

                if (path) {
                    result.next(path);
                    result.complete();
                    modal.close();
                }

                return false;
            },
            mcOnCancel: () => {
                result.complete();
                modal.close();
            }
        });

        return result;
    }

    private warnAboutDeleting(path: string) {
        const modal: McModalRef<DeleteFileModalComponent> = this.modalService.create({
            mcTitle: this.transloco.translate('modules.Modules.ModuleEdit.ModalTitle.RemoveMsgTitle'),
            mcSize: ModalSize.Small,
            mcComponent: DeleteFileModalComponent,
            mcComponentParams: {
                path
            },
            mcOkText: this.transloco.translate('common.Common.Pseudo.ButtonText.Delete'),
            mcCancelText: this.transloco.translate('common.Common.Pseudo.ButtonText.Cancel'),
            mcOnOk: () => {
                modal.close(true);
            },
            mcOnCancel: () => {
                modal.close(false);
            }
        });

        return modal.afterClose;
    }

    private showDeleteError() {
        this.toastService.show({
            style: 'error',
            title: this.transloco.translate('modules.Modules.ModuleEdit.ErrorText.FileRemoveError')
        });
    }

    private showMoveError() {
        this.toastService.show({
            style: 'error',
            title: this.transloco.translate('modules.Modules.ModuleEdit.ErrorText.FileMoveError')
        });
    }

    private showCreateError() {
        this.toastService.show({
            style: 'error',
            title: this.transloco.translate('modules.Modules.ModuleEdit.ErrorText.FileCreateError')
        });
    }

    private showDownloadError() {
        this.toastService.show({
            style: 'error',
            title: this.transloco.translate('modules.Modules.ModuleEdit.ErrorText.DownloadFileError')
        });
    }

    private showUploadError() {
        this.toastService.show({
            style: 'error',
            title: this.transloco.translate('modules.Modules.ModuleEdit.ErrorText.FileUploadError')
        });
    }
}
