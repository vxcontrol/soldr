import {
    Component,
    EventEmitter,
    Inject,
    Input,
    OnChanges,
    OnDestroy,
    OnInit,
    Output,
    SimpleChanges
} from '@angular/core';
import { TranslocoService } from '@ngneat/transloco';
import { ThemePalette } from '@ptsecurity/mosaic/core';
import { McModalRef, McModalService, ModalSize } from '@ptsecurity/mosaic/modal';
import { combineLatestWith, pairwise, Subject, takeUntil } from 'rxjs';

import { ModelsModuleS } from '@soldr/api';
import { PERMISSIONS_TOKEN } from '@soldr/core';
import { ModalInfoService, ModuleVersionPipe, parseErrorCode, ProxyPermission } from '@soldr/shared';
import { ModuleListFacade } from '@soldr/store/modules';

@Component({
    selector: 'soldr-delete-module',
    templateUrl: './delete-module.component.html',
    styleUrls: ['./delete-module.component.scss'],
    providers: [ModuleVersionPipe]
})
export class DeleteModuleComponent implements OnInit, OnChanges, OnDestroy {
    @Input() selectedModule: ModelsModuleS;
    @Input() compact: boolean;
    @Input() availableVersions: string[];

    @Output() afterDeleteVersion = new EventEmitter();
    @Output() afterDeleteModule = new EventEmitter();

    selectedVersion: string;
    themePalette = ThemePalette;
    tplModal: McModalRef;
    readonly destroyed$: Subject<void> = new Subject();

    constructor(
        private modalInfoService: ModalInfoService,
        private modalService: McModalService,
        private moduleVersionPipe: ModuleVersionPipe,
        private modulesFacade: ModuleListFacade,
        private transloco: TranslocoService,
        @Inject(PERMISSIONS_TOKEN) public permitted: ProxyPermission
    ) {}

    get hasMultipleVersions(): boolean {
        return this.availableVersions?.length > 1;
    }

    ngOnInit() {
        this.modulesFacade.isDeletingModule$
            .pipe(pairwise(), combineLatestWith(this.modulesFacade.deleteError$), takeUntil(this.destroyed$))
            .subscribe(([[previous, current], deleteError]) => {
                if (previous && !current) {
                    if (deleteError) {
                        this.showErrorModal(deleteError.code);
                    } else {
                        this.afterDeleteModule.emit();
                    }
                }
            });

        this.modulesFacade.isDeletingModuleVersion$
            .pipe(pairwise(), combineLatestWith(this.modulesFacade.deleteVersionError$), takeUntil(this.destroyed$))
            .subscribe(([[previous, current], deleteVersionError]) => {
                if (previous && !current) {
                    if (deleteVersionError) {
                        this.showErrorModal(deleteVersionError.code);
                    } else {
                        this.afterDeleteVersion.emit();
                    }
                }
            });
    }

    ngOnChanges({ selectedModule }: SimpleChanges) {
        if (selectedModule?.currentValue) {
            this.selectedVersion = this.moduleVersionPipe.transform(this.selectedModule.info.version);
        }
    }

    showErrorModal(code: string) {
        const errorText = this.transloco.translate(`modules.Modules.DeleteModule.ErrorText.${parseErrorCode(code)}`);
        this.modalInfoService.openErrorInfoModal(errorText);
    }

    ngOnDestroy(): void {
        this.destroyed$.next();
        this.destroyed$.complete();
    }

    confirmDeleteVersion(isVersionSelected = false) {
        this.tplModal = this.modalService.create({
            mcSize: ModalSize.Small,
            mcTitle: isVersionSelected
                ? this.transloco.translate('modules.Modules.DeleteModule.ModalTitle.DeleteSelectedVersion', {
                      version: this.selectedVersion
                  })
                : this.transloco.translate('modules.Modules.DeleteModule.ModalTitle.DeleteModule'),
            mcContent: isVersionSelected
                ? this.transloco.translate('modules.Modules.DeleteModule.Text.DeleteSelectedVersion', {
                      version: this.selectedVersion
                  })
                : this.transloco.translate('modules.Modules.DeleteModule.Text.DeleteModule'),
            mcOkText: isVersionSelected
                ? this.transloco.translate('modules.Modules.DeleteModule.ButtonText.DeleteSelectedVersion', {
                      version: this.selectedVersion
                  })
                : this.transloco.translate('modules.Modules.DeleteModule.ButtonText.DeleteModule'),
            mcCancelText: this.transloco.translate('common.Common.Pseudo.ButtonText.Cancel'),
            mcOnOk: () => (isVersionSelected ? this.deleteVersion() : this.delete()),
            mcOnCancel: () => this.destroyTplModal()
        });
    }

    destroyTplModal() {
        this.tplModal.destroy();
    }

    private deleteVersion() {
        this.destroyTplModal();
        this.modulesFacade.deleteModuleVersion(this.selectedModule.info.name, this.selectedVersion);
    }

    private delete() {
        this.destroyTplModal();
        this.modulesFacade.deleteModule(this.selectedModule.info.name);
    }
}
